// Demo tab switching
document.addEventListener('DOMContentLoaded', () => {
    // Tab functionality
    const tabs = document.querySelectorAll('.demo-tab');
    const panels = document.querySelectorAll('.demo-panel');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const targetId = `panel-${tab.dataset.tab}`;

            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            // Update active panel
            panels.forEach(p => p.classList.remove('active'));
            document.getElementById(targetId).classList.add('active');
        });
    });

    // Copy to clipboard functionality
    const copyButtons = document.querySelectorAll('.copy-btn');
    copyButtons.forEach(btn => {
        btn.addEventListener('click', async () => {
            const text = btn.dataset.copy;
            try {
                await navigator.clipboard.writeText(text);
                const originalText = btn.textContent;
                btn.textContent = 'Copied!';
                btn.style.color = 'var(--success)';
                setTimeout(() => {
                    btn.textContent = originalText;
                    btn.style.color = '';
                }, 2000);
            } catch (err) {
                console.error('Failed to copy:', err);
            }
        });
    });

    // Hero terminal typing animation
    animateHeroTerminal();

    // Smooth scroll for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
});

// Hero terminal typing animation
async function animateHeroTerminal() {
    const terminal = document.getElementById('hero-terminal');
    if (!terminal) return;

    const lines = terminal.querySelectorAll('.terminal-line, .terminal-output');

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];

        if (line.classList.contains('terminal-line')) {
            // Show the line
            line.classList.remove('hidden');
            line.classList.add('fade-in');

            // Type the command
            const command = line.querySelector('.command');
            if (command) {
                const text = command.dataset.typed;
                command.textContent = '';

                // Add cursor
                const cursor = document.createElement('span');
                cursor.className = 'cursor';
                command.appendChild(cursor);

                // Type each character
                for (let j = 0; j < text.length; j++) {
                    await sleep(40 + Math.random() * 30);
                    command.insertBefore(document.createTextNode(text[j]), cursor);
                }

                // Remove cursor after typing
                await sleep(200);
                cursor.remove();
            }

            await sleep(300);
        } else if (line.classList.contains('terminal-output')) {
            // Show output with fade in
            await sleep(200);
            line.classList.remove('hidden');
            line.classList.add('fade-in');
            await sleep(800);
        }
    }
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

// Intersection Observer for scroll animations
const observerOptions = {
    threshold: 0.1,
    rootMargin: '0px 0px -50px 0px'
};

const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.classList.add('fade-in');
            observer.unobserve(entry.target);
        }
    });
}, observerOptions);

// Observe elements for scroll animations
document.querySelectorAll('.feature-card, .orbit-card, .install-card').forEach(el => {
    el.style.opacity = '0';
    el.style.transform = 'translateY(20px)';
    observer.observe(el);
});

// Add scroll-based navbar styling
let lastScroll = 0;
window.addEventListener('scroll', () => {
    const navbar = document.querySelector('.navbar');
    const currentScroll = window.pageYOffset;

    if (currentScroll > 100) {
        navbar.style.background = 'rgba(10, 10, 15, 0.95)';
    } else {
        navbar.style.background = 'rgba(10, 10, 15, 0.8)';
    }

    lastScroll = currentScroll;
});
