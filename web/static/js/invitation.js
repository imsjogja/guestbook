/**
 * GuestFlow - Invitation Microsite JavaScript
 * Handles countdown, RSVP form, wish submission, and UI interactions.
 */
(function() {
    'use strict';

    // ==================== COUNTDOWN ====================
    const countdownEl = document.querySelector('.countdown');
    if (countdownEl) {
        const targetDate = new Date(countdownEl.dataset.target);
        const daysEl = document.getElementById('days');
        const hoursEl = document.getElementById('hours');
        const minutesEl = document.getElementById('minutes');
        const secondsEl = document.getElementById('seconds');

        function updateCountdown() {
            const now = new Date();
            const diff = targetDate - now;

            if (diff <= 0) {
                if (daysEl) daysEl.textContent = '00';
                if (hoursEl) hoursEl.textContent = '00';
                if (minutesEl) minutesEl.textContent = '00';
                if (secondsEl) secondsEl.textContent = '00';
                return;
            }

            const days = Math.floor(diff / (1000 * 60 * 60 * 24));
            const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
            const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
            const seconds = Math.floor((diff % (1000 * 60)) / 1000);

            if (daysEl) daysEl.textContent = String(days).padStart(2, '0');
            if (hoursEl) hoursEl.textContent = String(hours).padStart(2, '0');
            if (minutesEl) minutesEl.textContent = String(minutes).padStart(2, '0');
            if (secondsEl) secondsEl.textContent = String(seconds).padStart(2, '0');
        }

        updateCountdown();
        setInterval(updateCountdown, 1000);
    }

    // ==================== RSVP FORM ====================
    const rsvpForm = document.getElementById('rsvpForm');
    if (rsvpForm) {
        const paxGroup = document.getElementById('paxGroup');
        const adultsChildren = document.getElementById('adultsChildren');
        const menuGroup = document.getElementById('menuGroup');

        // Toggle fields based on attendance status
        rsvpForm.querySelectorAll('input[name="status"]').forEach(radio => {
            radio.addEventListener('change', function() {
                const isAttending = this.value === 'attending' || this.value === 'maybe';
                if (paxGroup) paxGroup.style.display = isAttending ? 'block' : 'none';
                if (adultsChildren) adultsChildren.style.display = isAttending ? 'flex' : 'none';
                if (menuGroup) menuGroup.style.display = isAttending ? 'block' : 'none';
            });
        });

        // Form submission
        rsvpForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            const submitBtn = document.getElementById('rsvpSubmit');
            const originalText = submitBtn.textContent;

            submitBtn.disabled = true;
            submitBtn.innerHTML = '<span class="loading"></span>';

            try {
                const formData = new FormData(rsvpForm);
                const data = Object.fromEntries(formData.entries());

                // Parse attending_pax as integer
                data.attending_pax = parseInt(data.attending_pax) || 1;
                data.adults = parseInt(data.adults) || 1;
                data.children = parseInt(data.children) || 0;

                // Parse attending_sessions if present
                if (data.attending_sessions) {
                    data.attending_sessions = [data.attending_sessions];
                }

                const response = await fetch('/api/v1/rsvp', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Accept': 'application/json'
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (response.ok) {
                    showToast(result.message || 'RSVP submitted successfully!', 'success');
                    // Reload page after short delay to show confirmation
                    setTimeout(() => window.location.reload(), 1500);
                } else {
                    showToast(result.error || 'Failed to submit RSVP. Please try again.', 'error');
                    submitBtn.disabled = false;
                    submitBtn.textContent = originalText;
                }
            } catch (err) {
                showToast('Network error. Please check your connection and try again.', 'error');
                submitBtn.disabled = false;
                submitBtn.textContent = originalText;
            }
        });
    }

    // ==================== WISH FORM ====================
    const wishForm = document.getElementById('wishForm');
    if (wishForm) {
        wishForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            const btn = wishForm.querySelector('button[type="submit"]');
            const originalText = btn.textContent;

            btn.disabled = true;
            btn.textContent = '...';

            // Simulate wish submission (endpoint not yet implemented)
            await new Promise(r => setTimeout(r, 500));
            showToast('Wish submitted! Thank you!', 'success');
            wishForm.reset();
            btn.disabled = false;
            btn.textContent = originalText;
        });
    }

    // ==================== TOAST NOTIFICATIONS ====================
    function showToast(message, type) {
        let toast = document.querySelector('.toast');
        if (!toast) {
            toast = document.createElement('div');
            toast.className = 'toast';
            document.body.appendChild(toast);
        }

        toast.textContent = message;
        toast.className = 'toast ' + (type || '');

        // Force reflow
        void toast.offsetWidth;

        toast.classList.add('show');

        setTimeout(() => {
            toast.classList.remove('show');
        }, 3000);
    }

    // ==================== SMOOTH SCROLL ====================
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        });
    });

    // ==================== INTERSECTION OBSERVER FOR ANIMATIONS ====================
    const observerOptions = {
        root: null,
        rootMargin: '0px',
        threshold: 0.1
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, observerOptions);

    document.querySelectorAll('.card, .timeline-item, .gallery-item').forEach(el => {
        el.style.opacity = '0';
        el.style.transform = 'translateY(20px)';
        el.style.transition = 'opacity 0.5s ease, transform 0.5s ease';
        observer.observe(el);
    });

})();
