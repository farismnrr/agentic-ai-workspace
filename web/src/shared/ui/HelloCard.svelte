<script lang="ts">
  interface Props {
    title: string;
    message: string;
  }

  let { title, message }: Props = $props();
</script>

<section class="card w-full max-w-2xl border border-base-content/10 bg-base-100 shadow-2xl">
  <div class="card-body gap-6 p-8 text-left sm:p-10">
    <div class="flex flex-wrap items-center gap-2">
      <span class="badge badge-primary">Masih Awam</span>
      <span class="badge badge-secondary badge-outline">AI workspace</span>
    </div>

    <div class="space-y-3">
      <p class="text-xs font-bold uppercase tracking-[0.14em] text-base-content/55">Design foundation</p>
      <h1 class="text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">{title}</h1>
      <p class="max-w-xl text-base leading-7 text-base-content/70">{message}</p>
    </div>

    <div class="flex flex-wrap gap-2">
      <button type="button" class="btn btn-primary">Primary</button>
      <button type="button" class="btn btn-secondary">Secondary</button>
      <button type="button" class="btn btn-accent">Tertiary</button>
      <button type="button" class="btn btn-ghost">Ghost</button>
    </div>
  </div>
</section>
