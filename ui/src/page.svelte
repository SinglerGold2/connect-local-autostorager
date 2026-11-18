<script lang="ts">
  import { Card, CardContent, PageHeader } from '@prusa/connect-ui/primitives';
  import { bind, type operations } from 'autostorager-client';

  import { heldHost } from './host';

  type Answer = operations['read-autostorager']['responses'][200]['content']['application/json'];

  const host = heldHost();
  const t = host.t;
  const routes = bind(host.client);

  let answer = $state<Answer | undefined>(undefined);

  $effect(() => {
    const run = async () => {
      try {
        answer = await routes.readAutostorager({}, host.signal);
      } catch {
        answer = undefined;
      }
    };

    void run();
    const timer = setInterval(run, 5000);
    host.signal.addEventListener('abort', () => clearInterval(timer));
  });
</script>

<div class="flex flex-col gap-4">
  <PageHeader title={t('autostorager.page.title')} />

  <Card>
    <CardContent>
      <div class="flex flex-col gap-2 text-sm">
        <p>{t('autostorager.page.what')}</p>
        <p class="text-neutral-600">{t('autostorager.page.delay')}</p>
        <p class="text-neutral-600">{t('autostorager.page.names')}</p>
        <p class="text-neutral-600">{t('autostorager.page.docker')}</p>
      </div>
    </CardContent>
  </Card>

  {#if answer !== undefined}
    <Card>
      <CardContent>
        <div class="grid grid-cols-2 gap-x-6 gap-y-1 text-sm">
          <span class="text-neutral-500">{t('autostorager.page.folder')}</span>
          <span>{answer.settings.folder || t('autostorager.page.none')}</span>
          <span class="text-neutral-500">{t('autostorager.page.interval')}</span>
          <span>{answer.settings.interval_seconds} s</span>
          <span class="text-neutral-500">{t('autostorager.page.storage')}</span>
          <span>{answer.settings.storage}</span>
          <span class="text-neutral-500">{t('autostorager.page.maxMb')}</span>
          <span>{answer.settings.max_mb} MB</span>
          <span class="text-neutral-500">{t('autostorager.page.moveOnSuccess')}</span>
          <span>{answer.settings.move_on_success ? t('autostorager.page.yes') : t('autostorager.page.no')}</span>
        </div>
      </CardContent>
    </Card>

    <Card>
      <CardContent>
        <h3 class="mb-2 font-medium">{t('autostorager.page.recent')}</h3>
        {#if answer.recent.length === 0}
          <p class="text-neutral-500 text-sm">{t('autostorager.page.nothingYet')}</p>
        {:else}
          <ul class="flex flex-col gap-1 text-sm">
            {#each answer.recent as entry (entry.at + entry.source)}
              <li class="border-neutral-200 flex flex-wrap items-center gap-2 border-t pt-1">
                <span class="text-neutral-500">{entry.at}</span>
                <span>{entry.source}</span>
                <span class="ml-auto">{entry.result}</span>
                {#if entry.detail}<span class="text-neutral-500">{entry.detail}</span>{/if}
              </li>
            {/each}
          </ul>
        {/if}
      </CardContent>
    </Card>
  {/if}
</div>
