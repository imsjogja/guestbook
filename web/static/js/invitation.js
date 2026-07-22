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

    // ==================== SELF CHECK-IN ====================
    const selfCheckinSection = document.getElementById('self-checkin');
    const selfCheckinStart = document.getElementById('selfCheckinStart');
    const selfCheckinPanel = document.getElementById('selfCheckinPanel');
    const selfCheckinVideo = document.getElementById('selfCheckinVideo');
    const selfCheckinCameraWrap = document.getElementById('selfCheckinCameraWrap');
    const selfCheckinStatus = document.getElementById('selfCheckinStatus');
    const selfCheckinSuccess = document.getElementById('selfCheckinSuccess');
    const selfCheckinSuccessMessage = document.getElementById('selfCheckinSuccessMessage');
    const selfCheckinManual = document.getElementById('selfCheckinManual');
    const selfCheckinStop = document.getElementById('selfCheckinStop');
    const selfCheckinManualSubmit = document.getElementById('selfCheckinManualSubmit');
    const selfCheckinEventToken = document.getElementById('selfCheckinEventToken');
    let selfCheckinStream = null;
    let selfCheckinControls = null;
    let selfCheckinScanning = false;
    let selfCheckinBusy = false;

    function setSelfCheckinStatus(message, type) {
        if (!selfCheckinStatus) return;
        selfCheckinStatus.textContent = message || '';
        selfCheckinStatus.className = 'self-checkin-status ' + (type || '');
    }

    function stopSelfCheckinCamera() {
        selfCheckinScanning = false;
        if (selfCheckinControls) {
            selfCheckinControls.stop();
            selfCheckinControls = null;
        }
        if (selfCheckinStream) {
            selfCheckinStream.getTracks().forEach(track => track.stop());
            selfCheckinStream = null;
        }
        if (selfCheckinVideo) {
            selfCheckinVideo.pause();
            selfCheckinVideo.srcObject = null;
        }
    }

    function showSelfCheckinSuccess(message) {
        stopSelfCheckinCamera();
        if (selfCheckinCameraWrap) selfCheckinCameraWrap.hidden = true;
        if (selfCheckinManual) selfCheckinManual.hidden = true;
        if (selfCheckinStop) selfCheckinStop.hidden = true;
        if (selfCheckinStatus) selfCheckinStatus.hidden = true;
        if (selfCheckinSuccessMessage && message) selfCheckinSuccessMessage.textContent = message;
        if (selfCheckinSuccess) selfCheckinSuccess.hidden = false;
        if (selfCheckinStart) selfCheckinStart.disabled = true;
    }

    function eventTokenFromValue(value) {
        const raw = String(value || '').trim();
        if (!raw) return '';
        try {
            const parsed = new URL(raw, window.location.origin);
            const match = parsed.pathname.match(/\/checkin\/event\/([^/]+)/i);
            if (match) return decodeURIComponent(match[1]);
        } catch (_) {
            // Treat a non-URL value as a raw event token below.
        }
        return raw;
    }

    async function submitSelfCheckin(value) {
        if (selfCheckinBusy || !selfCheckinSection) return;
        const eventToken = eventTokenFromValue(value);
        const invitationToken = selfCheckinSection.dataset.invitationToken || '';
        if (!eventToken || !invitationToken) {
            setSelfCheckinStatus('QR undangan tidak valid. Silakan buka kembali tautan undangan.', 'error');
            return;
        }

        selfCheckinBusy = true;
        setSelfCheckinStatus('Memproses check-in...', 'is-loading');
        try {
            const response = await fetch(selfCheckinSection.dataset.api || '/api/v1/self-checkin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
                body: JSON.stringify({ invitation_token: invitationToken, event_token: eventToken, actual_pax: 1 })
            });
            const result = await response.json().catch(() => ({}));
            if (response.ok) {
                showSelfCheckinSuccess('Check-in berhasil. Selamat datang di acara!');
                return;
            }
            if (response.status === 409) {
                showSelfCheckinSuccess('Tamu ini sudah tercatat check-in sebelumnya.');
                return;
            }
            setSelfCheckinStatus(result.error || 'Check-in gagal. Pastikan QR acara sesuai undangan ini.', 'error');
        } catch (_) {
            setSelfCheckinStatus('Tidak dapat terhubung ke server. Periksa koneksi lalu coba lagi.', 'error');
        } finally {
            selfCheckinBusy = false;
        }
    }

    async function startSelfCheckinCamera() {
        if (!selfCheckinPanel || !selfCheckinVideo) return;
        if (selfCheckinScanning) return;
        selfCheckinPanel.hidden = false;
        setSelfCheckinStatus('Meminta akses kamera...', 'is-loading');
        if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
            setSelfCheckinStatus('Kamera tidak didukung browser ini. Gunakan input token QR di bawah.', 'error');
            return;
        }
        try {
            selfCheckinScanning = true;
            setSelfCheckinStatus('Arahkan kamera ke QR acara.', '');
            if ('BarcodeDetector' in window) {
                selfCheckinStream = await navigator.mediaDevices.getUserMedia({
                    video: { facingMode: { ideal: 'environment' } },
                    audio: false
                });
                selfCheckinVideo.srcObject = selfCheckinStream;
                await selfCheckinVideo.play();
                const detector = new window.BarcodeDetector({ formats: ['qr_code'] });
                const scan = async () => {
                    if (!selfCheckinScanning || !selfCheckinVideo || selfCheckinVideo.readyState < 2) return;
                    try {
                        const codes = await detector.detect(selfCheckinVideo);
                        if (codes.length > 0 && codes[0].rawValue) {
                            await submitSelfCheckin(codes[0].rawValue);
                            return;
                        }
                    } catch (_) {
                        // Keep scanning; transient camera frames can fail detection.
                    }
                    if (selfCheckinScanning) window.setTimeout(scan, 250);
                };
                void scan();
                return;
            }

            if (window.ZXingBrowser && window.ZXingBrowser.BrowserQRCodeReader) {
                const reader = new window.ZXingBrowser.BrowserQRCodeReader();
                selfCheckinControls = await reader.decodeFromVideoDevice(
                    undefined,
                    selfCheckinVideo,
                    (result) => {
                        if (result && result.getText()) {
                            void submitSelfCheckin(result.getText());
                        }
                    }
                );
                return;
            }

            selfCheckinScanning = false;
            setSelfCheckinStatus('Scanner QR tidak tersedia. Gunakan input token QR di bawah.', 'error');
        } catch (err) {
            stopSelfCheckinCamera();
            const message = err && err.name === 'NotAllowedError'
                ? 'Akses kamera ditolak. Izinkan kamera atau gunakan input token QR di bawah.'
                : 'Kamera tidak dapat dibuka. Gunakan input token QR di bawah.';
            setSelfCheckinStatus(message, 'error');
        }
    }

    if (selfCheckinStart) selfCheckinStart.addEventListener('click', startSelfCheckinCamera);
    if (selfCheckinStop) selfCheckinStop.addEventListener('click', () => {
        stopSelfCheckinCamera();
        if (selfCheckinPanel) selfCheckinPanel.hidden = true;
        setSelfCheckinStatus('', '');
    });
    if (selfCheckinManualSubmit && selfCheckinEventToken) {
        selfCheckinManualSubmit.addEventListener('click', () => submitSelfCheckin(selfCheckinEventToken.value));
        selfCheckinEventToken.addEventListener('keydown', event => {
            if (event.key === 'Enter') {
                event.preventDefault();
                void submitSelfCheckin(selfCheckinEventToken.value);
            }
        });
    }
    window.addEventListener('pagehide', stopSelfCheckinCamera);

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
