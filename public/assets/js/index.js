// Buffkit Default JavaScript Entry Point
// This file serves as the main entry point for application JavaScript.
// It sets up core functionality and initializes components.

console.log('[Buffkit] Initializing application...');

// Import and initialize htmx extensions if needed
if (typeof htmx !== 'undefined') {
    // Configure htmx
    htmx.config.defaultSwapStyle = 'outerHTML';
    htmx.config.defaultSettleDelay = 20;
    htmx.config.defaultSwapDelay = 0;
    htmx.config.historyCacheSize = 0;

    // Log htmx events in development
    if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
        document.body.addEventListener('htmx:load', function(evt) {
            console.log('[htmx] Content loaded:', evt.detail.elt);
        });

        document.body.addEventListener('htmx:afterSwap', function(evt) {
            console.log('[htmx] Content swapped:', evt.detail.target);
        });
    }

    // Handle CSRF tokens for htmx requests
    document.body.addEventListener('htmx:configRequest', function(evt) {
        // Get CSRF token from meta tag
        const token = document.querySelector('meta[name="csrf-token"]');
        if (token) {
            evt.detail.headers['X-CSRF-Token'] = token.content;
        }
    });

    // Handle htmx errors
    document.body.addEventListener('htmx:responseError', function(evt) {
        console.error('[htmx] Request failed:', evt.detail);

        // Show user-friendly error message
        const target = evt.detail.target;
        if (target) {
            const errorHtml = `
                <div class="alert alert-error" role="alert">
                    <strong>Error:</strong> Failed to load content. Please try again.
                </div>
            `;

            // Insert error message before the target element
            target.insertAdjacentHTML('beforebegin', errorHtml);

            // Auto-remove error after 5 seconds
            setTimeout(() => {
                const alert = target.previousElementSibling;
                if (alert && alert.classList.contains('alert-error')) {
                    alert.remove();
                }
            }, 5000);
        }
    });
}

// Initialize Alpine.js components if available
if (typeof Alpine !== 'undefined') {
    // Register global Alpine data/stores here
    Alpine.store('buffkit', {
        initialized: true,
        version: '0.1.0-alpha',

        // Global state that components can access
        user: null,
        notifications: [],

        // Methods
        addNotification(message, type = 'info') {
            const notification = {
                id: Date.now(),
                message,
                type,
                timestamp: new Date()
            };

            this.notifications.push(notification);

            // Auto-remove after 5 seconds
            setTimeout(() => {
                this.removeNotification(notification.id);
            }, 5000);
        },

        removeNotification(id) {
            const index = this.notifications.findIndex(n => n.id === id);
            if (index > -1) {
                this.notifications.splice(index, 1);
            }
        }
    });

    // Register reusable Alpine components
    Alpine.data('dropdown', () => ({
        open: false,

        toggle() {
            this.open = !this.open;
        },

        close() {
            this.open = false;
        },

        init() {
            // Close on click outside
            this.$watch('open', (value) => {
                if (value) {
                    this.$nextTick(() => {
                        const handler = (e) => {
                            if (!this.$el.contains(e.target)) {
                                this.close();
                                document.removeEventListener('click', handler);
                            }
                        };
                        document.addEventListener('click', handler);
                    });
                }
            });
        }
    }));

    Alpine.data('modal', () => ({
        open: false,

        show() {
            this.open = true;
            document.body.style.overflow = 'hidden';
        },

        hide() {
            this.open = false;
            document.body.style.overflow = '';
        },

        toggle() {
            this.open ? this.hide() : this.show();
        }
    }));

    Alpine.data('tabs', () => ({
        activeTab: 0,

        selectTab(index) {
            this.activeTab = index;
        },

        isActive(index) {
            return this.activeTab === index;
        }
    }));
}

// Utility functions
window.Buffkit = window.Buffkit || {};

Object.assign(window.Buffkit, {
    // Flash message helper
    flash(message, type = 'info') {
        const container = document.querySelector('.flash-messages');
        if (!container) {
            console.warn('[Buffkit] Flash message container not found');
            return;
        }

        const messageEl = document.createElement('div');
        messageEl.className = `flash-message flash-${type}`;
        messageEl.textContent = message;
        messageEl.setAttribute('role', 'alert');

        container.appendChild(messageEl);

        // Auto-remove after 5 seconds
        setTimeout(() => {
            messageEl.remove();
        }, 5000);
    },

    // Confirmation dialog helper
    confirm(message, callback) {
        if (window.confirm(message)) {
            callback();
        }
    },

    // Form validation helper
    validateForm(formEl) {
        const inputs = formEl.querySelectorAll('[required]');
        let valid = true;

        inputs.forEach(input => {
            if (!input.value.trim()) {
                input.classList.add('error');
                valid = false;
            } else {
                input.classList.remove('error');
            }
        });

        return valid;
    },

    // AJAX helper (for non-htmx requests)
    ajax(url, options = {}) {
        const defaults = {
            method: 'GET',
            headers: {
                'X-Requested-With': 'XMLHttpRequest'
            }
        };

        // Add CSRF token if needed
        const token = document.querySelector('meta[name="csrf-token"]');
        if (token && options.method !== 'GET') {
            defaults.headers['X-CSRF-Token'] = token.content;
        }

        return fetch(url, { ...defaults, ...options })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response;
            });
    },

    // Debounce helper
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    // Throttle helper
    throttle(func, limit) {
        let inThrottle;
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }
});

// Auto-initialize components on DOM ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeComponents);
} else {
    initializeComponents();
}

function initializeComponents() {
    console.log('[Buffkit] Components initialized');

    // Initialize tooltips
    initTooltips();

    // Initialize form enhancements
    initForms();

    // Initialize keyboard shortcuts
    initKeyboardShortcuts();

    // Dispatch custom event
    document.dispatchEvent(new CustomEvent('buffkit:ready'));
}

function initTooltips() {
    // Simple CSS-only tooltips via data attributes
    document.querySelectorAll('[data-tooltip]').forEach(el => {
        el.setAttribute('title', el.dataset.tooltip);
        el.classList.add('has-tooltip');
    });
}

function initForms() {
    // Auto-submit forms with data-auto-submit
    document.querySelectorAll('form[data-auto-submit]').forEach(form => {
        form.addEventListener('change', () => {
            if (typeof htmx !== 'undefined' && form.hasAttribute('hx-post')) {
                htmx.trigger(form, 'submit');
            } else {
                form.submit();
            }
        });
    });

    // Confirmation for destructive actions
    document.querySelectorAll('[data-confirm]').forEach(el => {
        el.addEventListener('click', (e) => {
            if (!window.confirm(el.dataset.confirm)) {
                e.preventDefault();
                e.stopPropagation();
                return false;
            }
        });
    });

    // Auto-focus first input in modals
    document.addEventListener('shown.bs.modal', (e) => {
        const input = e.target.querySelector('input:not([type="hidden"]), textarea');
        if (input) {
            input.focus();
        }
    });
}

function initKeyboardShortcuts() {
    document.addEventListener('keydown', (e) => {
        // Ctrl/Cmd + K for search
        if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
            e.preventDefault();
            const searchInput = document.querySelector('[data-search-input]');
            if (searchInput) {
                searchInput.focus();
            }
        }

        // Escape to close modals/dropdowns
        if (e.key === 'Escape') {
            // Close dropdowns
            document.querySelectorAll('[data-dropdown].open').forEach(el => {
                el.classList.remove('open');
            });

            // Close modals
            document.querySelectorAll('.modal.show').forEach(el => {
                el.classList.remove('show');
            });
        }
    });
}

// Export for ES modules
export default window.Buffkit;
