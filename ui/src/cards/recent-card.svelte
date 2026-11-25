<script lang="ts">
  import type { CardProps } from '@prusa/connect-ui/slots';
  import { bind, type operations } from 'autostorager-client';

  import { heldHost } from '../host';
  import { applyLanguage, t } from '../i18n.svelte';

  type Answer =
    operations['read-autostorager']['responses'][200]['content']['application/json'];

  let { title }: CardProps = $props();

  const host = heldHost();
  const routes = bind(host.client);

  const heading = $derived(
    title.trim() === '' ? t('autostorager.card.heading') : title,
  );

  let answer = $state<Answer | undefined>(undefined);

  $effect(() => {
    const load = async () => {
      try {
        answer = await routes.readAutostorager({}, host.signal);
        applyLanguage(answer.language);
      } catch {
        answer = undefined;
      }
    };

    void load();
    const timer = setInterval(() => void load(), 10000);
    host.signal.addEventListener('abort', () => clearInterval(timer));
  });

  const lines = $derived((answer?.recent ?? []).slice(-10).reverse());
  const downSources = $derived(
    (answer?.sources ?? []).filter(
      (source) => source.state === 'down' || source.state === 'error',
    ),
  );

  function sourceLabel(state: string) {
    return state === 'down'
      ? t('autostorager.page.sourceDown')
      : t('autostorager.page.sourceError');
  }

  function nameOf(source: string): string {
    const parts = source.split(/[\\/]/);

    return parts[parts.length - 1] ?? source;
  }
</script>

<div class="flex h-full flex-col gap-2">
  <h3 class="text-base font-semibold">{heading}</h3>

  {#if downSources.length > 0}
    <ul class="flex flex-col gap-1 overflow-y-auto text-sm">
      {#each downSources as source (source.folder)}
        <li class="flex items-center gap-2">
          <span
            class="size-2 shrink-0 rounded-full"
            class:bg-state-attention-edge={source.state === 'down'}
            class:bg-state-fault={source.state === 'error'}
            role="img"
            aria-label={sourceLabel(source.state)}
          ></span>
          <span class="truncate">{nameOf(source.folder)}</span>
          <span class="text-neutral-500 ml-auto shrink-0"
            >{sourceLabel(source.state)}</span
          >
        </li>
      {/each}
    </ul>
  {/if}

  {#if lines.length === 0 && downSources.length === 0}
    <p class="text-neutral-500 text-sm">{t('autostorager.card.empty')}</p>
  {:else}
    <ul class="flex flex-col gap-1 overflow-y-auto text-sm">
      {#each lines as line (line.at + line.source)}
        <li class="flex items-center gap-2">
          <span
            class="size-2 shrink-0 rounded-full bg-state-done-edge"
            role="img"
            aria-label={t('autostorager.card.imported')}
          ></span>
          <span class="truncate">{nameOf(line.source)}</span>
          <span class="text-neutral-500 ml-auto shrink-0"
            >{line.at.slice(11, 16)}</span
          >
        </li>
      {/each}
    </ul>
  {/if}
</div>
