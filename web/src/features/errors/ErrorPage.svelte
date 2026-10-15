<script lang="ts">
  import { navigate } from '../../app/router';
  import { nextPath } from '../../shared/session/next-path';
  import { session, signOut } from '../../shared/session/session';
  import BrandLink from '../../shared/ui/BrandLink.svelte';
  import Illustration from '../../shared/ui/Illustration.svelte';
  import Marker from '../../shared/ui/Marker.svelte';
  import { errorCopy, type ErrorKind } from './content';

  interface Props {
    kind: ErrorKind;
  }

  const { kind }: Props = $props();

  const copy = $derived(errorCopy[kind]);
  const signedIn = $derived($session.status === 'signed-in');
  const homeHref = $derived(signedIn ? '/app' : '/');

  async function signInAgain(): Promise<void> {
    if (signedIn) await signOut();
    navigate(`/login?next=${encodeURIComponent(nextPath(window.location.search))}`);
  }

  function goBack(): void {
    if (window.history.length > 1) window.history.back();
    else navigate(homeHref);
  }
</script>

<svelte:head>
  <title>{copy.title} · Masih Awam</title>
  <meta name="robots" content="noindex" />
</svelte:head>

<div class="flex min-h-dvh flex-col bg-base-100 px-6 py-8 sm:px-12 lg:px-[clamp(3rem,8vw,7.5rem)]">
  <BrandLink />
  <main class="hero-in mx-auto my-auto flex w-full max-w-xl flex-col items-center py-10 text-center">
    <Illustration asset={copy.illustration} class="h-auto max-h-[40vh] w-auto max-w-full" priority />
    <p class="mt-8 text-sm text-base-content/60">{copy.label}</p>
    <h1 class="mt-2 text-balance text-5xl font-medium leading-[1.05] tracking-tight">
      {copy.before} <Marker>{copy.marked}</Marker>
    </h1>
    <p class="mt-4 max-w-md text-balance text-[var(--brand-text-secondary)]">{copy.body}</p>
    <div class="mt-8 flex flex-wrap justify-center gap-3">
      <a class="btn btn-primary min-h-12 rounded-full" href={homeHref}>{copy.home}</a>
      <button class="btn btn-outline min-h-12 rounded-full" type="button" onclick={kind === 'forbidden' ? signInAgain : goBack}>
        {signedIn ? copy.second : copy.secondAnonymous}
      </button>
    </div>
  </main>
</div>
