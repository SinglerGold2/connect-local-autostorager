<script lang="ts">
  import {
    Button,
    Card,
    CardContent,
    Input,
    Label,
    PageHeader,
    Switch,
  } from '@prusa/connect-ui/primitives';
  import { bind, type operations } from 'autostorager-client';

  import { heldHost } from './host';
  import { applyLanguage, t } from './i18n.svelte';

  type Answer =
    operations['read-autostorager']['responses'][200]['content']['application/json'];
  type Settings =
    operations['read-settings']['responses'][200]['content']['application/json'];

  const host = heldHost();
  const routes = bind(host.client);

  let answer = $state<Answer | undefined>(undefined);
  let draft = $state<Settings | undefined>(undefined);
  let busy = $state(false);
  let notice = $state('');

  $effect(() => {
    void load();
    const timer = setInterval(() => void load(), 10000);
    host.signal.addEventListener('abort', () => clearInterval(timer));
  });

  async function load() {
    try {
      answer = await routes.readAutostorager({}, host.signal);
      applyLanguage(answer.language);

      if (draft === undefined) {
        const loaded = await routes.readSettings({}, host.signal);
        draft = {
          ...loaded,
          folders: loaded.folders.length > 0 ? loaded.folders : [''],
        };
      }
    } catch {
      answer = undefined;
    }
  }

  async function save() {
    if (draft === undefined) return;

    busy = true;
    notice = '';

    try {
      const sent = {
        ...draft,
        folders: draft.folders.filter((folder) => folder.trim() !== ''),
      };
      draft = await routes.saveSettings({ body: sent }, host.signal);
      notice = t('autostorager.page.saved');
      await load();
    } catch (error) {
      notice = String(error);
    } finally {
      busy = false;
    }
  }

  function setFolder(index: number, value: string) {
    if (draft === undefined) return;

    draft.folders = draft.folders.map((folder, at) =>
      at === index ? value : folder,
    );
  }

  function addFolder() {
    if (draft === undefined || draft.folders.length >= draft.max_folders)
      return;

    draft.folders = [...draft.folders, ''];
  }

  function removeFolder(index: number) {
    if (draft === undefined) return;

    const rest = draft.folders.filter((_, at) => at !== index);
    draft.folders = rest.length > 0 ? rest : [''];
  }

  function setIntervalSeconds(value: string) {
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

  function sourceLabel(state: string) {
    switch (state) {
      case 'online':
        return t('autostorager.page.sourceOnline');
      case 'down':
        return t('autostorager.page.sourceDown');
      case 'starting':
        return t('autostorager.page.sourceStarting');
      default:
        return t('autostorager.page.sourceError');
    }
  }
</script>

<div class="flex flex-col gap-4">
  <PageHeader title={t('autostorager.page.title')} />

  <Card>
    <CardContent>
      <div class="flex flex-col gap-2">
        <p class="text-base">{t('autostorager.page.what')}</p>
        <p class="text-neutral-600 text-sm">{t('autostorager.page.delay')}</p>
        <p
          class="rounded border-amber-500 bg-amber-500/10 border-l-4 p-2 text-sm"
        >
          {t('autostorager.page.readManual')}
        </p>
      </div>
    </CardContent>
  </Card>

  {#if draft !== undefined}
    <Card>
      <CardContent>
        <div class="flex flex-col gap-3">
          <div class="flex flex-col gap-2">
            <Label>{t('autostorager.page.folders')}</Label>

            {#each draft.folders as folder, index (index)}
              <div class="flex items-center gap-2">
                <Input
                  value={folder}
                  placeholder={t('autostorager.page.folderPlaceholder')}
                  class="w-full"
                  oninput={(event: Event) => {
                    const target =
                      event.currentTarget as HTMLInputElement | null;
                    if (target !== null) setFolder(index, target.value);
                  }}
                />
                {#if draft.folders.length > 1}
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={busy}
                    onclick={() => removeFolder(index)}
                  >
                    {t('autostorager.page.remove')}
                  </Button>
                {/if}
              </div>
            {/each}

            <div class="flex items-center gap-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={busy || draft.folders.length >= draft.max_folders}
                onclick={addFolder}
              >
                {t('autostorager.page.addFolder')}
              </Button>
              <span class="text-neutral-500 text-xs"
                >{draft.folders.length} / {draft.max_folders}</span
              >
            </div>
            <p class="text-neutral-500 text-xs">
              {t('autostorager.page.folderHint')}
            </p>
            <p class="text-neutral-500 text-xs">
              {t('autostorager.page.maxFoldersHint')}
            </p>
          </div>

          <div class="flex flex-col gap-1">
            <Label for="ast-interval">{t('autostorager.page.interval')}</Label>
            <Input
              value={String(draft.interval_seconds)}
              class="w-32"
              oninput={(event: Event) => {
                const target = event.currentTarget as HTMLInputElement | null;
                if (target !== null) setIntervalSeconds(target.value);
              }}
            />
            <p class="text-neutral-500 text-xs">
              {t('autostorager.page.intervalHint')}
            </p>
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
            <p class="text-neutral-500 text-xs">
              {t('autostorager.page.maxHint')}
            </p>
          </div>

          <div class="flex items-center gap-2">
            <Switch checked={draft.move_on_success} onCheckedChange={setMove} />
            <span class="text-sm">{t('autostorager.page.moveOnSuccess')}</span>
          </div>

          <p class="text-neutral-500 text-xs">{t('autostorager.page.names')}</p>
          <div class="flex items-center gap-2">
            <Button disabled={busy} onclick={save}
              >{t('autostorager.page.save')}</Button
            >
            {#if notice !== ''}<span class="text-sm">{notice}</span>{/if}
          </div>
        </div>
      </CardContent>
    </Card>
  {/if}

  {#if answer !== undefined}
    {#if answer.sources.length > 0}
      <Card>
        <CardContent>
          <h3 class="mb-2 font-medium">{t('autostorager.page.sources')}</h3>
          <ul class="flex flex-col gap-1 text-sm">
            {#each answer.sources as source (source.folder)}
              <li
                class="border-neutral-200 flex flex-wrap items-center gap-2 border-t pt-1"
              >
                <span
                  class="size-2 shrink-0 rounded-full"
                  class:bg-state-done-edge={source.state === 'online'}
                  class:bg-state-attention-edge={source.state === 'down'}
                  class:bg-state-fault={source.state === 'error'}
                  class:bg-neutral-400={source.state === 'starting'}
                  role="img"
                  aria-label={sourceLabel(source.state)}
                ></span>
                <span class="truncate">{source.folder}</span>
                <span class="text-neutral-500 ml-auto"
                  >{sourceLabel(source.state)}</span
                >
                {#if source.detail}
                  <span class="text-neutral-500 w-full text-xs"
                    >{source.detail}</span
                  >
                {/if}
              </li>
            {/each}
          </ul>
        </CardContent>
      </Card>
    {/if}

    <Card>
      <CardContent>
        <h3 class="mb-2 font-medium">{t('autostorager.page.recent')}</h3>
        {#if answer.recent.length === 0}
          <p class="text-neutral-500 text-sm">
            {t('autostorager.page.nothingYet')}
          </p>
        {:else}
          <ul class="flex flex-col gap-1 text-sm">
            {#each answer.recent as entry (entry.at + entry.source)}
              <li
                class="border-neutral-200 flex flex-wrap items-center gap-2 border-t pt-1"
              >
                <span class="text-neutral-500">{entry.at}</span>
                <span class="truncate">{entry.source}</span>
                <span class="ml-auto">{entry.result}</span>
                {#if entry.detail}<span class="text-neutral-500"
                    >{entry.detail}</span
                  >{/if}
              </li>
            {/each}
          </ul>
        {/if}
      </CardContent>
    </Card>

    {#if answer.logs.length > 0}
      <Card>
        <CardContent>
          <h3 class="mb-2 font-medium">{t('autostorager.page.logs')}</h3>
          <ul class="flex flex-col gap-1 text-sm">
            {#each answer.logs as entry (entry.at + entry.source + (entry.detail ?? ''))}
              <li
                class="border-neutral-200 flex flex-wrap items-center gap-2 border-t pt-1"
              >
                <span class="text-neutral-500">{entry.at}</span>
                <span class="truncate">{entry.source}</span>
                <span class="ml-auto">{entry.result}</span>
                {#if entry.detail}<span class="text-neutral-500"
                    >{entry.detail}</span
                  >{/if}
              </li>
            {/each}
          </ul>
        </CardContent>
      </Card>
    {/if}
  {/if}
</div>
