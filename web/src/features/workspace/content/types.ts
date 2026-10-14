export type Tone = 'primary' | 'secondary' | 'accent';

export interface Project {
  id: string;
  name: string;
  subtitle: string;
  tone: Tone;
  updated: string;
  site: string;
  featured?: boolean;
}

export interface Rule {
  id: string;
  text: string;
  on: boolean;
}
