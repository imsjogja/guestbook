const fs = require('fs');
const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');

const W = 1600;
const H = 900;
const C = {
  ink: '#14213d',
  muted: '#65708a',
  paper: '#f7f3eb',
  white: '#fffdf8',
  coral: '#f26b5b',
  mint: '#35b7a7',
  mintDark: '#168d83',
  blue: '#6f8cff',
  yellow: '#f6c85f',
  line: '#d9d7d0',
};

function esc(value) {
  return String(value).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function text(x, y, value, size = 24, options = {}) {
  const fill = options.fill || C.ink;
  const weight = options.weight >= 700 ? 'bold' : 'normal';
  const anchor = options.anchor || 'start';
  const spacing = options.spacing || 0;
  return `<text x="${x}" y="${y}" fill="${fill}" font-family="Helvetica, Arial, sans-serif" font-size="${size}px" font-weight="${weight}" text-anchor="${anchor}" letter-spacing="${spacing}px">${esc(value)}</text>`;
}

function textLines(x, y, values, size = 24, leading = 32, options = {}) {
  return values.map((value, index) => text(x, y + index * leading, value, size, options)).join('');
}

function box(x, y, width, height, fill = C.white, radius = 22, shadow = false) {
  return `<rect x="${x}" y="${y}" width="${width}" height="${height}" rx="${radius}" fill="${fill}"${shadow ? ' filter="url(#shadow)"' : ''}/>`;
}

function outline(x, y, width, height, stroke = C.line, radius = 22) {
  return `<rect x="${x}" y="${y}" width="${width}" height="${height}" rx="${radius}" fill="none" stroke="${stroke}" stroke-width="2"/>`;
}

function circle(x, y, radius, fill, extra = '') {
  return `<circle cx="${x}" cy="${y}" r="${radius}" fill="${fill}" ${extra}/>`;
}

function line(x1, y1, x2, y2, stroke = C.line, width = 2, extra = '') {
  return `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="${stroke}" stroke-width="${width}" ${extra}/>`;
}

function arrow(x1, y1, x2, y2, stroke = C.coral, width = 4) {
  return line(x1, y1, x2, y2, stroke, width, 'marker-end="url(#arrow)"');
}

function pill(x, y, label, fill, width) {
  return box(x, y, width, 34, fill, 17) + text(x + width / 2, y + 23, label, 13, { fill: C.ink, weight: 800, anchor: 'middle', spacing: 1 });
}

function check(x, y, color = C.mint) {
  return circle(x, y, 13, color) + `<path d="M ${x - 6} ${y} l 4 4 l 8 -9" fill="none" stroke="${C.white}" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>`;
}

function qr(x, y, size) {
  const cell = size / 21;
  let output = box(x - 16, y - 16, size + 32, size + 32, C.white, 14);
  const finder = (fr, fc) => {
    let result = '';
    for (let row = 0; row < 7; row += 1) {
      for (let col = 0; col < 7; col += 1) {
        const edge = row === 0 || row === 6 || col === 0 || col === 6;
        const core = row >= 2 && row <= 4 && col >= 2 && col <= 4;
        if (edge || core) result += box(x + (fc + col) * cell, y + (fr + row) * cell, cell + 0.4, cell + 0.4, C.ink, 0);
      }
    }
    return result;
  };
  for (let row = 0; row < 21; row += 1) {
    for (let col = 0; col < 21; col += 1) {
      const finderCell = (row < 7 && col < 7) || (row < 7 && col >= 14) || (row >= 14 && col < 7);
      if (!finderCell && ((row * 17 + col * 29 + row * col * 7) % 11 < 5 || (row + col) % 9 === 0)) {
        output += box(x + col * cell, y + row * cell, cell + 0.4, cell + 0.4, C.ink, 0);
      }
    }
  }
  return output + finder(0, 0) + finder(0, 14) + finder(14, 0);
}

function base(content, page, section) {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">
    <defs>
      <linearGradient id="hero" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${C.coral}"/><stop offset="1" stop-color="${C.yellow}"/></linearGradient>
      <marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="5" orient="auto"><path d="M 0 0 L 10 5 L 0 10 z" fill="${C.coral}"/></marker>
      <filter id="shadow" x="-30%" y="-30%" width="160%" height="160%"><feDropShadow dx="0" dy="12" stdDeviation="14" flood-color="${C.ink}" flood-opacity="0.14"/></filter>
    </defs>
    <rect width="${W}" height="${H}" fill="${C.paper}"/>
    <circle cx="1530" cy="60" r="150" fill="${C.yellow}" opacity="0.20"/>
    <circle cx="80" cy="850" r="130" fill="${C.mint}" opacity="0.12"/>
    ${text(74, 64, 'GUESTFLOW  /  PARTNERSHIP PITCH', 14, { fill: C.muted, weight: 800, spacing: 2 })}
    ${text(1525, 64, String(page).padStart(2, '0'), 15, { fill: C.muted, weight: 800, anchor: 'end', spacing: 2 })}
    ${text(74, 858, section.toUpperCase(), 13, { fill: C.muted, weight: 800, spacing: 2 })}
    ${content}
  </svg>`;
}

function heading(title, subtitle) {
  return text(74, 155, title, 50, { weight: 800, spacing: -1.2 }) + text(74, 198, subtitle, 20, { fill: C.muted });
}

function slide1() {
  let c = pill(74, 138, 'PARTNERSHIP', C.coral, 130);
  c += textLines(74, 252, ['GuestFlow untuk', 'vendor undangan digital'], 68, 76, { weight: 800, spacing: -2 });
  c += text(78, 452, 'Kelola tamu dari undangan sampai check-in.', 27, { fill: C.muted, weight: 700 });
  c += box(78, 548, 420, 74, C.ink, 37);
  c += text(288, 595, 'API  +  WEBHOOK  +  OPERASIONAL TAMU', 15, { fill: C.white, weight: 800, anchor: 'middle', spacing: 0.7 });
  c += circle(1200, 420, 250, C.coral, 'opacity="0.12"');
  c += circle(1200, 420, 185, C.yellow, 'opacity="0.30"');
  c += box(1050, 190, 300, 480, C.white, 24, true);
  c += box(1070, 210, 260, 78, C.ink, 18);
  c += text(1092, 242, 'VENDOR INVITATION', 12, { fill: C.yellow, weight: 800, spacing: 1.5 });
  c += text(1092, 270, 'Bambang & Rina', 23, { fill: C.white, weight: 800 });
  c += text(1092, 330, 'RSVP CONFIRMED', 13, { fill: C.mintDark, weight: 800, spacing: 1.5 });
  c += text(1092, 367, 'Guest pass siap dipakai', 20, { weight: 800 });
  c += qr(1114, 410, 170);
  c += text(1199, 620, 'SCAN DI VENUE', 12, { fill: C.muted, weight: 800, anchor: 'middle', spacing: 2 });
  c += text(1405, 760, 'Vendor tetap fokus pada undangan. GuestFlow mengurus tamunya.', 17, { fill: C.muted, weight: 700, anchor: 'end' });
  return base(c, 1, 'The idea');
}

function slide2() {
  let c = heading('Hari ini data tamu terpecah.', 'Vendor punya undangan. Tim acara masih bekerja manual.');
  const cards = [
    ['1', 'Daftar tamu', 'Tersimpan di spreadsheet, chat, atau form yang berbeda.', C.coral],
    ['2', 'RSVP', 'Jawaban hadir belum otomatis menjadi roster operasional.', C.yellow],
    ['3', 'Check-in', 'Petugas mencari nama satu per satu saat acara dimulai.', C.blue],
  ];
  cards.forEach((item, index) => {
    const x = 74 + index * 493;
    c += box(x, 300, 430, 270, C.white, 22, true);
    c += circle(x + 52, 358, 25, item[3]);
    c += text(x + 52, 364, item[0], 14, { weight: 800, anchor: 'middle' });
    c += text(x + 34, 428, item[1], 28, { weight: 800 });
    c += textLines(x + 34, 470, item[2].match(/.{1,39}(?:\s|$)/g).map((value) => value.trim()).filter(Boolean), 17, 28, { fill: C.muted });
    c += text(x + 34, 540, 'GAP OPERASIONAL', 12, { fill: item[3], weight: 800, spacing: 1.4 });
  });
  c += arrow(270, 670, 1330, 670, C.coral, 4);
  c += text(800, 718, 'Peluang: satukan event, tamu, RSVP, QR, dan kehadiran.', 24, { weight: 800, anchor: 'middle' });
  return base(c, 2, 'The problem');
}

function slide3() {
  let c = heading('Satu alur end-to-end.', 'Bukan hanya QR. Seluruh data tamu ikut terhubung.');
  const steps = [
    ['01', 'Buat event', 'Vendor membuat acara', C.coral],
    ['02', 'Sync tamu', 'Nama, kontak, tipe, plus-one', C.yellow],
    ['03', 'RSVP', 'Jawaban tamu tersimpan', C.mint],
    ['04', 'QR & check-in', 'GuestFlow validasi di venue', C.blue],
    ['05', 'Laporan hadir', 'Vendor menerima status aktual', C.coral],
  ];
  c += line(155, 440, 1445, 440, C.line, 6);
  steps.forEach((step, index) => {
    const x = 180 + index * 310;
    c += circle(x, 440, 35, step[3]);
    c += text(x, 448, step[0], 15, { weight: 800, anchor: 'middle' });
    c += text(x, 345, step[1], 23, { weight: 800, anchor: 'middle' });
    c += textLines(x - 112, 500, step[2].match(/.{1,22}(?:\s|$)/g).map((value) => value.trim()).filter(Boolean), 16, 25, { fill: C.muted, anchor: 'middle' });
    if (index < steps.length - 1) c += arrow(x + 42, 440, x + 265, 440, C.coral, 3);
  });
  c += box(280, 690, 1040, 68, C.ink, 34);
  c += text(800, 733, 'Event  ->  Guest roster  ->  RSVP  ->  QR  ->  Check-in  ->  Report', 20, { fill: C.white, weight: 800, anchor: 'middle' });
  return base(c, 3, 'The flow');
}

function slide4() {
  let c = heading('Apa yang diintegrasikan?', 'Vendor dan GuestFlow berbagi data sesuai peran masing-masing.');
  c += box(74, 275, 600, 410, C.white, 24, true);
  c += pill(112, 315, 'VENDOR  ->  GUESTFLOW', C.coral, 220);
  const inbound = ['Event dan jadwal', 'Daftar tamu', 'Perubahan profil tamu', 'Jawaban RSVP'];
  inbound.forEach((value, index) => { c += check(128, 402 + index * 52, C.coral); c += text(164, 409 + index * 52, value, 21, { weight: 700 }); });
  c += box(926, 275, 600, 410, C.ink, 24, true);
  c += pill(964, 315, 'GUESTFLOW  ->  VENDOR', C.yellow, 220);
  const outbound = ['Status RSVP tersinkron', 'QR per tamu', 'Status check-in', 'Rekap kehadiran'];
  outbound.forEach((value, index) => { c += check(980, 402 + index * 52, C.mint); c += text(1016, 409 + index * 52, value, 21, { fill: C.white, weight: 700 }); });
  c += box(710, 405, 180, 150, 'url(#hero)', 26, true);
  c += text(800, 455, 'API', 30, { weight: 800, anchor: 'middle' });
  c += text(800, 497, '+', 28, { weight: 800, anchor: 'middle' });
  c += text(800, 535, 'WEBHOOK', 18, { weight: 800, anchor: 'middle', spacing: 0.7 });
  c += arrow(674, 480, 710, 480, C.coral, 3);
  c += arrow(890, 480, 926, 480, C.coral, 3);
  c += text(800, 755, 'Vendor tetap memiliki pengalaman undangan. GuestFlow menjadi operational layer tamu.', 22, { weight: 800, anchor: 'middle' });
  return base(c, 4, 'The integration');
}

function slide5() {
  let c = heading('GuestFlow masuk ke menu tamu vendor.', 'Bukan halaman terpisah. Data tamu dapat dipakai kembali di seluruh flow.');
  c += box(90, 270, 770, 430, C.white, 24, true);
  c += box(90, 270, 770, 72, C.ink, 24);
  c += box(90, 315, 770, 27, C.ink, 0);
  c += text(125, 315, 'VENDOR DASHBOARD  /  DAFTAR TAMU', 14, { fill: C.yellow, weight: 800, spacing: 1.2 });
  c += text(125, 385, 'Tamu acara', 28, { weight: 800 });
  c += pill(622, 362, '+ Tambah tamu', C.coral, 145);
  c += text(125, 432, 'Nama', 14, { fill: C.muted, weight: 800 });
  c += text(410, 432, 'RSVP', 14, { fill: C.muted, weight: 800 });
  c += text(545, 432, 'QR', 14, { fill: C.muted, weight: 800 });
  c += text(670, 432, 'Check-in', 14, { fill: C.muted, weight: 800 });
  const rows = [['Bambang Kusniawan', 'Hadir', 'Aktif', 'Belum'], ['Rina Pratiwi', 'Hadir', 'Aktif', 'Sudah'], ['Keluarga Santoso', 'Menunggu', '-', '-']];
  rows.forEach((row, index) => {
    const y = 478 + index * 58;
    c += line(125, y - 25, 825, y - 25, C.line, 1);
    c += text(125, y, row[0], 17, { weight: 700 });
    c += pill(390, y - 22, row[1], row[1] === 'Hadir' ? C.mint : C.yellow, row[1] === 'Menunggu' ? 100 : 72);
    c += text(555, y, row[2], 16, { fill: row[2] === 'Aktif' ? C.mintDark : C.muted, weight: 800 });
    c += text(680, y, row[3], 16, { fill: row[3] === 'Sudah' ? C.mintDark : C.muted, weight: 800 });
  });
  c += box(980, 280, 440, 380, C.ink, 24, true);
  c += pill(1020, 320, 'GUESTFLOW OPERATIONS', C.mint, 190);
  c += text(1020, 400, 'Di belakang layar:', 23, { fill: C.white, weight: 800 });
  ['Roster per event', 'QR per tamu', 'Validasi scan', 'Audit dan laporan'].forEach((value, index) => { c += check(1032, 455 + index * 45, C.mint); c += text(1068, 462 + index * 45, value, 18, { fill: '#d8deef', weight: 600 }); });
  c += text(800, 758, 'Satu data tamu dapat dipakai untuk RSVP, komunikasi, seating, dan check-in.', 22, { weight: 800, anchor: 'middle' });
  return base(c, 5, 'The product');
}

function slide6() {
  let c = heading('Manfaat untuk vendor dan pemilik acara.', 'Vendor menambah nilai. Pemilik acara mendapat kontrol operasional.');
  c += box(74, 285, 690, 390, C.ink, 24, true);
  c += pill(112, 325, 'VENDOR', C.yellow, 110);
  c += text(112, 415, 'Produk lebih lengkap.', 32, { fill: C.white, weight: 800 });
  ['Fitur guest management siap pakai', 'Paket premium lebih kuat', 'Tidak perlu membangun scanner sendiri', 'Revenue share atau add-on'].forEach((value, index) => { c += check(124, 485 + index * 42, C.yellow); c += text(158, 492 + index * 42, value, 18, { fill: '#d8deef', weight: 600 }); });
  c += box(836, 285, 690, 390, C.white, 24, true);
  c += pill(874, 325, 'PEMILIK ACARA', C.mint, 165);
  c += text(874, 415, 'Tamu lebih terkontrol.', 32, { weight: 800 });
  ['Satu daftar tamu per acara', 'RSVP dan check-in lebih akurat', 'QR otomatis tanpa kerja manual', 'Laporan hadir setelah acara'].forEach((value, index) => { c += check(886, 485 + index * 42, C.mint); c += text(920, 492 + index * 42, value, 18, { fill: C.muted, weight: 600 }); });
  c += box(310, 745, 980, 60, 'url(#hero)', 30);
  c += text(800, 783, 'Vendor menjual pengalaman. GuestFlow memastikan operasionalnya berjalan.', 20, { weight: 800, anchor: 'middle' });
  return base(c, 6, 'The value');
}

function slide7() {
  let c = pill(74, 138, 'PILOT 30 HARI', C.coral, 145);
  c += textLines(74, 258, ['Mulai dari', 'satu vendor.', 'Satu event.'], 66, 75, { weight: 800, spacing: -2 });
  c += textLines(78, 520, ['Kita buktikan alur lengkap:', 'event -> tamu -> RSVP -> QR -> check-in -> laporan.'], 23, 36, { fill: C.muted, weight: 700 });
  c += box(78, 660, 390, 64, C.ink, 32);
  c += text(273, 701, 'PARTNER PILOT', 17, { fill: C.white, weight: 800, anchor: 'middle', spacing: 1.2 });
  c += circle(1210, 425, 250, C.coral, 'opacity="0.12"');
  c += circle(1210, 425, 185, C.yellow, 'opacity="0.28"');
  c += box(1055, 215, 310, 430, C.white, 24, true);
  c += box(1075, 235, 270, 70, C.ink, 18);
  c += text(1097, 265, 'GUESTFLOW', 13, { fill: C.yellow, weight: 800, spacing: 1.8 });
  c += text(1097, 290, 'PARTNER PILOT', 20, { fill: C.white, weight: 800 });
  ['Connect event', 'Sync guest list', 'Show QR on RSVP', 'Check-in live'].forEach((value, index) => { c += check(1107, 370 + index * 48, C.mint); c += text(1142, 377 + index * 48, value, 19, { weight: 700 }); });
  c += line(1097, 560, 1322, 560, C.line, 2);
  c += text(1097, 603, 'Ready untuk partner pertama.', 17, { fill: C.muted, weight: 700 });
  c += text(74, 820, 'GuestFlow  |  guestflow.id', 15, { fill: C.muted, weight: 800, spacing: 1 });
  return base(c, 7, 'The next step');
}

const slides = [slide1(), slide2(), slide3(), slide4(), slide5(), slide6(), slide7()];
const tempDir = process.env.PITCHDECK_ASSET_DIR || fs.mkdtempSync(path.join(os.tmpdir(), 'guestflow-pitchdeck-'));
fs.mkdirSync(tempDir, { recursive: true });
const svgFiles = slides.map((svg, index) => {
  const filename = path.join(tempDir, `slide-${String(index + 1).padStart(2, '0')}.svg`);
  fs.writeFileSync(filename, svg);
  return filename;
});
const output = path.join(__dirname, 'pitchdeck_guestflow_vendor_partnership.pdf');
const rsvg = process.env.RSVG_BIN || '/opt/homebrew/opt/librsvg/bin/rsvg-convert';
const pdfFiles = svgFiles.map((svgFile) => {
  const pdfFile = svgFile.replace(/\.svg$/, '.pdf');
  execFileSync(rsvg, ['-f', 'pdf', '-o', pdfFile, svgFile], { stdio: 'inherit' });
  return pdfFile;
});
const mergeScript = [
  'from pypdf import PdfWriter',
  'import sys',
  'writer = PdfWriter()',
  '[writer.append(filename) for filename in sys.argv[1:-1]]',
  'output_file = open(sys.argv[-1], "wb")',
  'writer.write(output_file)',
  'output_file.close()',
].join('; ');
execFileSync('python3', ['-c', mergeScript, ...pdfFiles, output], { stdio: 'inherit' });
console.log(`Created ${output}`);
