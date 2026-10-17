// Runs inside the page (it is sent to the browser as text), so it may use nothing but the page's own globals.
export function inspectPage() {
  const visible = (el: Element) => {
    const box = el.getBoundingClientRect();
    const style = getComputedStyle(el);
    return box.width > 0 && box.height > 0 && style.visibility !== 'hidden' && style.display !== 'none';
  };
  const channel = (value: string) => value.match(/[\d.]+/g)?.map(Number) ?? [0, 0, 0, 1];
  const luminance = ([r = 0, g = 0, b = 0]: number[]) => {
    const [rl = 0, gl = 0, bl = 0] = [r, g, b].map((c) => {
      const s = c / 255;
      return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
    });
    return 0.2126 * rl + 0.7152 * gl + 0.0722 * bl;
  };
  const backdrop = (el: Element | null): number[] => {
    for (let node = el; node; node = node.parentElement) {
      const c = channel(getComputedStyle(node).backgroundColor);
      if ((c[3] ?? 1) > 0.9) return c;
    }
    return [255, 255, 255, 1];
  };
  const lowContrast: string[] = [];
  const texts = [...document.querySelectorAll('h1,h2,h3,h4,p,a,li,button,label,span,td,th')]
    .filter((el) => visible(el) && [...el.childNodes].some((n) => n.nodeType === 3 && (n.textContent ?? '').trim() !== ''))
    .slice(0, 400);
  for (const el of texts) {
    const style = getComputedStyle(el);
    const fg = luminance(channel(style.color)), bg = luminance(backdrop(el));
    const ratio = (Math.max(fg, bg) + 0.05) / (Math.min(fg, bg) + 0.05);
    const large = parseFloat(style.fontSize) >= 24 || (parseFloat(style.fontSize) >= 18.66 && Number(style.fontWeight) >= 700);
    if (ratio < (large ? 3 : 4.5) && lowContrast.length < 8) {
      lowContrast.push(`${el.tagName.toLowerCase()} "${(el.textContent ?? '').trim().slice(0, 30)}" ${ratio.toFixed(1)}:1`);
    }
  }
  const images = [...document.images];
  const targets = [...document.querySelectorAll('a[href],button,input,select,textarea,[role=button]')].filter(visible);
  const unlabeled = [...document.querySelectorAll('input:not([type=hidden]),select,textarea')].filter((el) => {
    const id = el.getAttribute('id');
    return !el.getAttribute('aria-label') && !el.getAttribute('aria-labelledby') && !el.closest('label') &&
      !(id && document.querySelector(`label[for="${id}"]`));
  });
  const body = getComputedStyle(document.body);
  return {
    title: document.title,
    lang: document.documentElement.lang,
    viewportMeta: document.querySelector('meta[name=viewport]') !== null,
    width: innerWidth,
    pageWidth: document.documentElement.scrollWidth,
    pageHeight: document.documentElement.scrollHeight,
    overflowX: document.documentElement.scrollWidth > innerWidth + 1,
    headings: [...document.querySelectorAll('h1,h2,h3')].filter(visible).slice(0, 30)
      .map((h) => `${h.tagName.toLowerCase()} ${(h.textContent ?? '').trim().slice(0, 60)}`),
    landmarks: ['header', 'nav', 'main', 'footer'].filter((tag) => document.querySelector(tag) !== null),
    images: { total: images.length, broken: images.filter((i) => i.complete && i.naturalWidth === 0).length,
      noAlt: images.filter((i) => !i.hasAttribute('alt')).length },
    smallTargets: targets.filter((el) => { const b = el.getBoundingClientRect(); return b.width < 24 || b.height < 24; }).length,
    unlabeledFields: unlabeled.length,
    font: body.fontFamily.slice(0, 80),
    lowContrast,
    text: document.body.innerText.replace(/\s+/g, ' ').trim().slice(0, 500),
  };
}
