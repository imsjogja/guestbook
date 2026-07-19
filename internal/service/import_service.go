// Package service provides business logic layer implementations for GuestFlow.
package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"guestflow/internal/domain"
	"guestflow/internal/repository"

	"github.com/google/uuid"
)

// ImportService handles CSV/Excel import operations for guests.
type ImportService struct {
	guestRepo *repository.GuestRepository
}

// NewImportService creates a new ImportService.
func NewImportService(guestRepo *repository.GuestRepository) *ImportService {
	return &ImportService{guestRepo: guestRepo}
}

// CSVTemplateHeaders returns the standard CSV template headers.
var CSVTemplateHeaders = []string{
	"full_name", "nickname", "phone", "email", "address", "city", "country",
	"guest_type", "segment", "institution", "title", "relationship", "pic",
	"accessibility_needs", "dietary_restrictions", "allergies", "notes",
}

// CSVTemplate returns the CSV template content as bytes.
func (s *ImportService) CSVTemplate() []byte {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	_ = writer.Write(CSVTemplateHeaders)
	writer.Flush()
	return buf.Bytes()
}

// ImportCSV parses CSV content and bulk imports guests.
// It supports UTF-8 BOM, comma and semicolon separators, and returns per-row error details.
func (s *ImportService) ImportCSV(ctx context.Context, tenantID, createdBy uuid.UUID, content []byte) (*domain.GuestImportResult, error) {
	return s.importCSV(ctx, tenantID, createdBy, content, false)
}

// ImportCSVForEvent imports guests for an event roster. Existing tenant guests
// are reused instead of being reported as duplicates; the event roster owns
// the event-specific association.
func (s *ImportService) ImportCSVForEvent(ctx context.Context, tenantID, createdBy uuid.UUID, content []byte) (*domain.GuestImportResult, error) {
	return s.importCSV(ctx, tenantID, createdBy, content, true)
}

func (s *ImportService) importCSV(ctx context.Context, tenantID, createdBy uuid.UUID, content []byte, reuseExisting bool) (*domain.GuestImportResult, error) {
	content = stripBOM(content)

	// Detect separator: try comma first, then semicolon
	reader := csv.NewReader(bytes.NewReader(content))
	reader.TrimLeadingSpace = true

	// Try to detect separator from first line
	firstLine, err := readFirstLine(content)
	if err == nil {
		if strings.Contains(firstLine, ";") && !strings.Contains(firstLine, ",") {
			reader.Comma = ';'
		}
	}
	reader.ReuseRecord = false

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	headerMap := normalizeHeaders(headers)
	if err := validateHeaders(headerMap); err != nil {
		return nil, err
	}

	// Parse all rows
	var rows []domain.GuestImportRow
	rowNum := 2 // Row 1 is header

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Log the error but continue; malformed rows are captured below
			rows = append(rows, domain.GuestImportRow{
				RowNum: rowNum,
				Errors: []string{fmt.Sprintf("Failed to parse row: %v", err)},
			})
			rowNum++
			continue
		}

		// Skip completely empty rows
		if isEmptyRow(record) {
			rowNum++
			continue
		}

		row := parseRow(record, headerMap, rowNum)
		rows = append(rows, row)
		rowNum++
	}

	// Validate each row and collect valid guests
	var validGuests []*domain.Guest
	var result domain.GuestImportResult
	result.TotalRows = len(rows)

	for i := range rows {
		validateRow(&rows[i])
		if len(rows[i].Errors) == 0 {
			guest := rowToGuest(&rows[i], tenantID, createdBy)
			validGuests = append(validGuests, guest)
		} else {
			result.ErrorCount++
			result.Errors = append(result.Errors, rows[i])
		}
	}

	// Check duplicates against existing database
	var reusedGuestIDs []uuid.UUID
	if len(validGuests) > 0 {
		var phones, emails []string
		for _, g := range validGuests {
			if g.Phone != nil && *g.Phone != "" {
				phones = append(phones, *g.Phone)
			}
			if g.Email != nil && *g.Email != "" {
				emails = append(emails, *g.Email)
			}
		}

		duplicates, err := s.guestRepo.CheckDuplicates(ctx, tenantID, phones, emails)
		if err != nil {
			return nil, fmt.Errorf("check duplicates during import: %w", err)
		}

		// Mark duplicates and filter them out
		var uniqueGuests []*domain.Guest
		for i, g := range validGuests {
			var dupErrors []string
			duplicateIDs := make(map[uuid.UUID]struct{})
			if g.Phone != nil && *g.Phone != "" {
				if dupID, ok := duplicates["phone:"+*g.Phone]; ok {
					dupErrors = append(dupErrors, fmt.Sprintf("Phone number already exists (guest ID: %s)", dupID))
					duplicateIDs[dupID] = struct{}{}
				}
			}
			if g.Email != nil && *g.Email != "" {
				if dupID, ok := duplicates["email:"+*g.Email]; ok {
					dupErrors = append(dupErrors, fmt.Sprintf("Email already exists (guest ID: %s)", dupID))
					duplicateIDs[dupID] = struct{}{}
				}
			}

			if len(dupErrors) > 0 {
				if reuseExisting && len(duplicateIDs) == 1 {
					for duplicateID := range duplicateIDs {
						reusedGuestIDs = append(reusedGuestIDs, duplicateID)
					}
					continue
				}
				// Find the corresponding row in results
				for j := range rows {
					if rows[j].RowNum == i+2 { // approximate mapping
						// Already validated, mark as duplicate
						break
					}
				}
				result.ErrorCount++
				// We need to find which row this guest maps to
				for j := range rows {
					if rows[j].FullName == g.FullName {
						if (g.Phone != nil && rows[j].Phone == *g.Phone) || (g.Email != nil && rows[j].Email == *g.Email) {
							rows[j].Errors = append(rows[j].Errors, dupErrors...)
							result.Errors = append(result.Errors, rows[j])
							break
						}
					}
				}
			} else {
				uniqueGuests = append(uniqueGuests, g)
			}
		}
		validGuests = uniqueGuests
	}

	// Check duplicates within the import itself
	seenPhones := make(map[string]bool)
	seenEmails := make(map[string]bool)
	var finalGuests []*domain.Guest

	for _, g := range validGuests {
		var dupErrors []string
		if g.Phone != nil && *g.Phone != "" {
			if seenPhones[*g.Phone] {
				dupErrors = append(dupErrors, "Duplicate phone number within import file")
			}
			seenPhones[*g.Phone] = true
		}
		if g.Email != nil && *g.Email != "" {
			if seenEmails[*g.Email] {
				dupErrors = append(dupErrors, "Duplicate email within import file")
			}
			seenEmails[*g.Email] = true
		}
		if len(dupErrors) == 0 {
			finalGuests = append(finalGuests, g)
		}
	}

	// Bulk insert valid guests
	if len(finalGuests) > 0 {
		if err := s.guestRepo.BulkCreate(ctx, finalGuests); err != nil {
			return nil, fmt.Errorf("bulk insert guests: %w", err)
		}
	}

	result.SuccessCount = len(finalGuests)
	if reuseExisting {
		result.SuccessCount += len(reusedGuestIDs)
	}
	result.ImportedGuestIDs = make([]uuid.UUID, 0, len(finalGuests)+len(reusedGuestIDs))
	for _, guest := range finalGuests {
		result.ImportedGuestIDs = append(result.ImportedGuestIDs, guest.ID)
	}
	if reuseExisting {
		result.ImportedGuestIDs = append(result.ImportedGuestIDs, reusedGuestIDs...)
	}
	return &result, nil
}

// stripBOM removes the UTF-8 Byte Order Mark if present.
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// readFirstLine returns the first line of the content for separator detection.
func readFirstLine(data []byte) (string, error) {
	idx := bytes.IndexByte(data, '\n')
	if idx == -1 {
		idx = len(data)
	}
	return string(data[:idx]), nil
}

// normalizeHeaders maps header names to their canonical form.
func normalizeHeaders(headers []string) map[string]int {
	canonical := make(map[string]int)
	for i, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		// Normalize common variations
		h = strings.ReplaceAll(h, " ", "_")
		h = strings.ReplaceAll(h, "-", "_")
		canonical[h] = i
	}
	return canonical
}

// validateHeaders checks that required headers are present.
func validateHeaders(headers map[string]int) error {
	if _, ok := headers["full_name"]; !ok {
		if _, ok := headers["name"]; !ok {
			if _, ok := headers["fullname"]; !ok {
				return fmt.Errorf("CSV header 'full_name' is required")
			}
		}
	}
	return nil
}

// parseRow maps a CSV record to a GuestImportRow using the header map.
func parseRow(record []string, headerMap map[string]int, rowNum int) domain.GuestImportRow {
	get := func(keys ...string) string {
		for _, key := range keys {
			if idx, ok := headerMap[key]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
		}
		return ""
	}

	return domain.GuestImportRow{
		RowNum:              rowNum,
		FullName:            get("full_name", "name", "fullname"),
		Nickname:            get("nickname", "nick_name"),
		Phone:               get("phone", "phone_number", "mobile"),
		Email:               get("email", "email_address"),
		Address:             get("address"),
		City:                get("city"),
		Country:             get("country"),
		GuestType:           get("guest_type", "type", "guesttype"),
		Segment:             get("segment", "category"),
		Institution:         get("institution", "organization", "company", "org"),
		Title:               get("title"),
		Relationship:        get("relationship"),
		PIC:                 get("pic", "person_in_charge"),
		AccessibilityNeeds:  get("accessibility_needs", "accessibility", "special_needs"),
		DietaryRestrictions: get("dietary_restrictions", "dietary", "diet"),
		Allergies:           get("allergies"),
		Notes:               get("notes", "remarks", "comments"),
	}
}

// validateRow validates a single import row, collecting all errors.
func validateRow(row *domain.GuestImportRow) {
	var errs []string

	// full_name is required
	if row.FullName == "" {
		errs = append(errs, "Full name is required")
	} else if utf8.RuneCountInString(row.FullName) < 2 {
		errs = append(errs, "Full name must be at least 2 characters")
	} else if len(row.FullName) > 255 {
		errs = append(errs, "Full name must be at most 255 characters")
	}

	// Validate email format if provided
	if row.Email != "" && !isValidEmail(row.Email) {
		errs = append(errs, "Invalid email format")
	}

	// Validate phone E.164 format if provided
	if row.Phone != "" && !isValidE164(row.Phone) {
		errs = append(errs, "Phone must be in E.164 format (e.g., +628123456789)")
	}

	// Validate guest_type if provided
	if row.GuestType != "" && !domain.IsValidGuestType(row.GuestType) {
		errs = append(errs, "Invalid guest_type. Must be one of: vip, vvip, family, friend, colleague, government, media, sponsor, vendor, speaker, participant, internal, protocol, security, general")
	}

	row.Errors = errs
}

// rowToGuest converts a validated import row to a domain Guest.
func rowToGuest(row *domain.GuestImportRow, tenantID, createdBy uuid.UUID) *domain.Guest {
	req := domain.GuestCreateRequest{
		FullName:             row.FullName,
		Nickname:             row.Nickname,
		Phone:                row.Phone,
		Email:                row.Email,
		Address:              row.Address,
		City:                 row.City,
		Country:              row.Country,
		GuestType:            row.GuestType,
		Segment:              row.Segment,
		Institution:          row.Institution,
		Title:                row.Title,
		Relationship:         row.Relationship,
		PIC:                  row.PIC,
		AccessibilityNeeds:   row.AccessibilityNeeds,
		DietaryRestrictions:  row.DietaryRestrictions,
		Allergies:            row.Allergies,
		Notes:                row.Notes,
		ConsentCommunication: false,
	}

	return domain.NewGuest(tenantID, createdBy, req)
}

// isEmptyRow checks if a CSV record is entirely empty.
func isEmptyRow(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}

// emailRegex is a simple but effective email validation pattern.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isValidEmail checks if the email format is valid.
func isValidEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// isValidE164 checks if the phone number is in E.164 format.
func isValidE164(phone string) bool {
	if len(phone) < 8 || len(phone) > 15 {
		return false
	}
	// Must start with + followed by digits only
	if phone[0] != '+' {
		return false
	}
	for i := 1; i < len(phone); i++ {
		if phone[i] < '0' || phone[i] > '9' {
			return false
		}
	}
	return true
}

// intPtr converts an int to *int.
func intPtr(i int) *int {
	return &i
}

// parseInt safely parses an integer from a string.
func parseInt(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}
	i, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return nil, err
	}
	return &i, nil
}
