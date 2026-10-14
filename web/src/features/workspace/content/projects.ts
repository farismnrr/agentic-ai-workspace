import type { Project } from './types';

export const projects: Project[] = [
  { id: 'bakery', name: 'Bakery website', subtitle: 'Sweet Crumbs', site: 'sweetcrumbs.id', tone: 'primary', updated: 'Today', featured: true },
  { id: 'shop', name: 'Online shop', subtitle: 'Batik store', site: 'batikku.id', tone: 'secondary', updated: 'Today' },
  { id: 'wedding', name: 'Wedding invitation', subtitle: 'Rina & Dimas', site: 'rinadimas.id', tone: 'accent', updated: 'Yesterday', featured: true },
  { id: 'portfolio', name: 'Portfolio', subtitle: 'Personal site', site: 'sari.id', tone: 'primary', updated: 'Oct 9' },
  { id: 'blog', name: 'Community blog', subtitle: 'Weekly stories', site: 'ceritakita.id', tone: 'secondary', updated: 'Oct 8' },
  { id: 'kids', name: 'Kids class booking', subtitle: 'Sign-up page', site: 'kelaskecil.id', tone: 'accent', updated: 'Oct 2' },
];

export const sidebarProjectIds = ['bakery', 'shop', 'wedding', 'portfolio'];

export const findProject = (id: string): Project | undefined => projects.find((project) => project.id === id);
