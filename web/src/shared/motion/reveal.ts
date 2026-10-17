import type { Action } from 'svelte/action';

let observer: IntersectionObserver | undefined;

function sharedObserver(): IntersectionObserver {
  observer ??= new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          entry.target.setAttribute('data-revealed', '');
          observer?.unobserve(entry.target);
        }
      }
    },
    { rootMargin: '0px 0px -5% 0px', threshold: 0 },
  );
  return observer;
}

/** Fades an element in once it first scrolls into view. Pair with the `.reveal` class. */
export const reveal: Action<HTMLElement, number | undefined> = (node, delay = 0) => {
  node.classList.add('reveal');
  node.style.setProperty('--reveal-delay', `${String(delay)}ms`);
  const io = sharedObserver();
  io.observe(node);

  return {
    destroy() {
      io.unobserve(node);
    },
  };
};
