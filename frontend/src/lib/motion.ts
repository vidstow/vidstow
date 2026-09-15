import { cubicOut } from 'svelte/easing';
import { slide } from 'svelte/transition';

export const MOTION_HOVER = 120;
export const MOTION_CLOSE = 150;
export const MOTION_OPEN = 250;
export const MOTION_PROGRESS = 180;

const DIALOG_SCALE_FROM = 0.96;

export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return true;
  if (import.meta.env?.MODE === 'test') return true;
  if (typeof window.matchMedia !== 'function') return true;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

export function motionMs(ms: number): number {
  return prefersReducedMotion() ? 0 : ms;
}

type MotionParams = { duration?: number };

export function fadeOverlay(_node: Element, params: MotionParams = {}) {
  return {
    duration: motionMs(params.duration ?? MOTION_OPEN),
    easing: cubicOut,
    css: (t: number) => `opacity: ${t}`,
  };
}

export function scaleDialog(_node: Element, params: MotionParams = {}) {
  return {
    duration: motionMs(params.duration ?? MOTION_OPEN),
    easing: cubicOut,
    css: (t: number) => `transform: scale(${DIALOG_SCALE_FROM + (1 - DIALOG_SCALE_FROM) * t})`,
  };
}

export function panelSlide(node: Element, params: MotionParams = {}) {
  return slide(node, { duration: motionMs(params.duration ?? MOTION_OPEN) });
}

export function slidingPill(node: HTMLElement, _selected: unknown) {
  node.classList.add('has-sliding-pill');
  const pill = document.createElement('span');
  pill.className = 'sliding-pill';
  pill.setAttribute('aria-hidden', 'true');
  node.prepend(pill);

  let placed = false;

  const place = () => {
    const selected = node.querySelector<HTMLElement>('.on, .active, [aria-pressed="true"], [aria-checked="true"]');
    if (!selected) {
      pill.style.opacity = '0';
      return;
    }

    const parent = node.getBoundingClientRect();
    const rect = selected.getBoundingClientRect();
    const styles = getComputedStyle(node);
    const left = rect.left - parent.left - parseFloat(styles.borderLeftWidth);
    const top = rect.top - parent.top - parseFloat(styles.borderTopWidth);

    if (selected.matches(':disabled') || selected.closest('[aria-disabled="true"]')) {
      node.style.setProperty('--pill-bg', 'var(--surface-raised)');
    } else if (selected.classList.contains('file')) {
      node.style.setProperty('--pill-bg', 'var(--accent-soft)');
    } else if (selected.classList.contains('active')) {
      node.style.setProperty('--pill-bg', 'var(--surface-base)');
    } else {
      node.style.setProperty('--pill-bg', 'var(--surface-raised)');
    }

    if (!placed) {
      pill.style.transition = 'none';
    }
    pill.style.width = `${rect.width}px`;
    pill.style.height = `${rect.height}px`;
    pill.style.transform = `translate(${left}px, ${top}px)`;
    pill.style.opacity = '1';
    if (!placed) {
      requestAnimationFrame(() => {
        pill.style.transition = '';
        placed = true;
      });
    }
  };

  const resize = typeof ResizeObserver === 'function' ? new ResizeObserver(place) : null;
  resize?.observe(node);
  for (const button of node.querySelectorAll('button')) resize?.observe(button);

  const mutations = typeof MutationObserver === 'function'
    ? new MutationObserver(place)
    : null;
  mutations?.observe(node, { attributes: true, subtree: true, attributeFilter: ['class', 'aria-pressed', 'aria-checked', 'disabled', 'aria-disabled'] });

  requestAnimationFrame(place);

  return {
    update() {
      requestAnimationFrame(place);
    },
    destroy() {
      resize?.disconnect();
      mutations?.disconnect();
      pill.remove();
      node.classList.remove('has-sliding-pill');
      node.style.removeProperty('--pill-bg');
    },
  };
}
