<script lang="ts">
  import { Button, Card, CardContent, Input, Label, PageHeader, Switch } from '@prusa/connect-ui/primitives';
  import { bind, type operations } from 'autostorager-client';

  import { heldHost } from './host';

  type Answer = operations['read-autostorager']['responses'][200]['content']['application/json'];
  type Settings = operations['read-settings']['responses'][200]['content']['application/json'];

  const host = heldHost();
  const t = host.t;
  const routes = bind(host.client);

  let answer = $state<Answer | undefined>(undefined);
  let draft = $state<Settings | undefined>(undefined);
  let busy = $state(false);
  let notice = $state('');

  $effect(() => {
    void load();
  });

  async function load() {
    try {
      answer = await routes.readAutostorager({}, host.signal);
      if (draft === undefined) draft = await routes.readSettings({}, host.signal);
    } catch {
      answer = undefined;
    }
  }

  async function save() {
    if (draft === undefined) return;

    busy = true;
    notice = '';

    try {
      draft = await routes.saveSettings({ body: draft }, host.signal);
      notice = t('autostorager.page.saved');
      await load();
    } catch (error) {
      notice = String(error);
    } finally {
      busy = false;
    }
  }

  function setFolder(value: string) {
    if (draft !== undefined) draft.folder = value;
  }

  function setInterval(value: string) {
    if (draft !== undefined) draft.interval_seconds = Number(value);
  }

  function setStorage(value: string) {
    if (draft !== undefined) draft.storage = value;
  }

  function setMaxMB(value: string) {
    if (draft !== undefined) draft.max_mb = Number(value);
  }

  function setMove(value: boolean) {
    if (draft !== undefined) draft.move_on_success = value;
  }
</script>

<div class="flex flex-col gap-4">
  <PageHeader title={t('autostorager.page.title')} />

  <Card>
    <CardContent>
      <div class="flex flex-col gap-2 text-sm">
        <p>{t('autostorager.page.what')}</p>
        <p class="text-neutral-600">{t('autostorager.page.delay')}</p>
        <p class="text-neutral-600">{t('autostorager.page.names')}</p>
        <p class="text-neutral-600">
          {t('autostorager.page.docker')}
          <a class="underline" href="https://github.com/singlergold2/connect-local-autostorager#docker" target="_blank" rel="noreferrer">
            {t('autostorager.page.dockerLink')}
          </a>
        </p>
      </div>
    </CardContent>
  </Card>

  {#if draft !== undefined}
    <Card>
      <CardContent>
        <div class="flex flex-col gap-3">
          <div class="flex flex-col gap-1">
            <Label for="ast-folder">{t('autostorager.page.folder')}</Label>
            <Input
              value={draft.folder}
              placeholder="/home/you/autostorager"
              class="w-full"
              oninput={(event: Event) => {
                const target = event.currentTarget as HTMLInputElement | null;
                if (target !== null) setFolder(target.value);
              }}
            />
            <p class="text-neutral-500 text-xs">{t('autostorager.page.folderHint')}</p>
          </div>

          <div class="flex flex-col gap-1">
            <Label for="ast-interval">{t('autostorager.page.interval')}</Label>
            <Input
              value={String(draft.interval_seconds)}
              class="w-32"
              oninput={(event: Event) => {
                const target = event.currentTarget as HTMLInputElement | null;
                if (target !== null) setInterval(target.value);
              }}
            />
            <p class="text-neutral-500 text-xs">{t('autostorager.page.intervalHint')}</p>
          </div>

          <div class="flex flex-col gap-1">
            <Label for="ast-storage">{t('autostorager.page.storage')}</Label>
            <Input
              value={draft.storage}
              class="w-48"
              oninput={(event: Event) => {
                const target = event.currentTarget as HTMLInputElement | null;
                if (target !== null) setStorage(target.value);
              }}
            />
          </div>

          <div class="flex flex-col gap-1">
            <Label for="ast-max">{t('autostorager.page.maxMb')}</Label>
            <Input
              value={String(draft.max_mb)}
              class="w-32"
              oninput={(event: Event) => {
                const target = event.currentTarget as HTMLInputElement | null;
                if (target !== null) setMaxMB(target.value);
              }}
            />
            <p class="text-neutral-500 text-xs">{t('autostorager.page.maxHint')}</p>
          </div>

          <div class="flex items-center gap-2">
            <Switch checked={draft.move_on_success} onCheckedChange={setMove} />
            <span class="text-sm">{t('autostorager.page.moveOnSuccess')}</span>
          </div>

          <div class="flex items-center gap-2">
            <Button disabled={busy} onclick={save}>{t('autostorager.page.save')}</Button>
            {#if notice !== ''}<span class="text-sm">{notice}</span>{/if}
          </div>
        </div>
      </CardContent>
    </Card>
  {/if}

  {#if answer !== undefined}
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
