import { createRepo } from './github-api';
import { applyTemplate, archiveProject, createProject, importRepo } from './projects-api';
import type { Project } from './task-types';

export interface NewProject {
  name: string;
  description: string;
  /** The project ID; it cannot be changed after the project is made. */
  slug: string;
  source: 'empty' | 'github' | 'new';
  /** '' means an empty project; otherwise the name of the template to start from. */
  template: string;
  repo: string;
  repoName: string;
  isPrivate: boolean;
}

/** Makes the project, then fills it. If filling fails the project is archived so nothing half-made is left. */
export async function createNewProject(input: NewProject): Promise<Project> {
  // A new repository is made first: if GitHub refuses, no project is left behind.
  const target = input.source === 'new' ? (await createRepo(input.repoName, input.isPrivate)).name : input.repo;
  const project = await createProject(input.name, input.description, input.slug);
  try {
    if (input.source !== 'empty') await importRepo(project.id, target);
    if (input.template !== '') await applyTemplate(project.id, input.template);
  } catch (failure) {
    await archiveProject(project.id).catch(() => undefined);
    throw failure;
  }
  return project;
}
