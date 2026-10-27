import type { IconName } from '../../../shared/ui/icon-names';
import { statusMeta } from '../content/status';
import type { Project, Task } from '../logic/task-types';

export interface Command {
  id: string;
  label: string;
  hint: string;
  href: string;
  icon: IconName;
}

const pages: Command[] = [
  { id: 'new', label: 'New task', hint: 'Action', href: '/app', icon: 'plus' },
  { id: 'home', label: 'Home', hint: 'Go to', href: '/app', icon: 'home' },
  { id: 'inbox', label: 'Inbox', hint: 'Go to', href: '/app/inbox', icon: 'inbox' },
  { id: 'projects', label: 'Projects', hint: 'Go to', href: '/app/projects', icon: 'folder' },
  { id: 'sites', label: 'Sites', hint: 'Go to', href: '/app/sites', icon: 'globe' },
  { id: 'archive', label: 'Archive', hint: 'Go to', href: '/app/archive', icon: 'box' },
  { id: 'extensions', label: 'Extensions', hint: 'Go to', href: '/app/extensions', icon: 'puzzle' },
  { id: 'settings', label: 'Settings', hint: 'Go to', href: '/app/settings', icon: 'settings' },
  { id: 'usage', label: 'Usage', hint: 'Settings', href: '/app/settings/usage', icon: 'clock' },
];

export const allCommands = (projects: Project[], tasks: Task[]): Command[] => [
  ...pages,
  ...projects.map((project): Command => ({ id: `p-${project.id}`, label: project.name, hint: 'Project', href: `/app/projects/${project.id}`, icon: 'folder' })),
  ...tasks.map(
    (task): Command => ({ id: `t-${task.id}`, label: task.title, hint: 'Task', href: `/app/projects/${task.projectId}/tasks/${task.id}`, icon: statusMeta[task.status].icon }),
  ),
];

export const filterCommands = (list: Command[], query: string): Command[] => {
  const needle = query.trim().toLowerCase();
  return (needle === '' ? list.slice(0, 8) : list.filter((item) => item.label.toLowerCase().includes(needle))).slice(0, 10);
};
