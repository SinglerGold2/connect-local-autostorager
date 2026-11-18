<script lang="ts">
  import type { CardProps } from '@prusa/connect-ui/slots';
  import { bind, type operations } from 'autostorager-client';

  import { heldHost } from '../host';

  type Answer =
    operations['read-autostorager']['responses'][200]['content']['application/json'];

  let { title }: CardProps = $props();

  const host = heldHost();
  const t = host.t;
  const routes = bind(host.client);

  let answer = $state<Answer | undefined>(undefined);

  $effect(() => {
    const load = async () => {
      try {
        answer = await routes.readAutostorager({}, host.signal);
      } catch {
        answer = undefined;
      }
    };

    void load();
    const timer = setInterval(load, 10000);
    host.signal.addEventListener('abort', () => clearInterval(timer));
  });

  const lines = $derived((answer?.recent ?? []).slice(-5).reverse());

  function nameOf(source: string): string {
    const parts = source.split(/[\\/]/);

    return parts[parts.length - 1] ?? source;
  }
</script>

<div class="flex h-full flex-col gap-2">
  <h3 class="font-medium">{title}</h3>

  {#if lines.length === 0}
    <p class="text-neutral-500 text-sm">{t('autostorager.card.empty')}</p>
  {:else}
    <ul class="flex flex-col gap-1 text-sm">
      {#each lines as line (line.at + line.source)}
        <li class="flex items-center gap-2">
          <span class="truncate">{nameOf(line.source)}</span>
          <span class="text-neutral-500 ml-auto shrink-0"
            >{line.at.slice(11, 16)}</span
          >
        </li>
      {/each}
    </ul>
  {/if}
</div>
