<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
</script>

<div class="alert alert-info items-start text-sm">
  <Icon name="info" class="mt-0.5 size-4" />
  <div>
    <p>
      Agent receives the connection address as a variable; direct connections from the sandbox
      are not supported.
    </p>
    <p class="mt-1">
      Private and LAN addresses work for connection tests. In Docker, use
      <code>host.docker.internal</code> or a LAN IP.
    </p>
  </div>
</div>
