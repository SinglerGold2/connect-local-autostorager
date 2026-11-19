<script lang="ts">
  import { Card, CardContent, PageHeader } from '@prusa/connect-ui/primitives';

  import step1 from './assets/1-export-from-slicer.webp';
  import step2 from './assets/2-save-into-the-share.webp';
  import step3 from './assets/3-enter-the-folder.webp';
  import step4 from './assets/4-it-appears-in-storage.webp';
  import step5 from './assets/5-with-its-slicer-details.webp';
  import hostSettings from './assets/host-settings.webp';

  import { Button } from '@prusa/connect-ui/primitives';

  import english from './manual/en.json';
  import german from './manual/de.json';
  import czech from './manual/cs.json';
  import { applyLanguage, currentTag } from './i18n.svelte';

  // One file per language under manual/, so a translator copies en.json and
  // works in their own. The page follows the extension's language setting; the
  // buttons switch it for this visit.
  const books = { en: english, de: german, cs: czech } as const;
  type Tag = keyof typeof books;
  const tags = Object.keys(books) as Tag[];

  const language = $derived<Tag>(currentTag());

  // The tag is what the browser reports; the button shows the country, which
  // is what a reader recognises.
  const labels: Record<Tag, string> = { en: 'EN', de: 'DE', cs: 'CZ' };

  function m(key: string): string {
    const book = books[language] as Record<string, string>;

    return book[key] ?? (english as Record<string, string>)[key] ?? key;
  }
</script>

<div class="flex flex-col gap-4">
  <PageHeader title={m('title')} />

  <div class="flex items-center gap-2">
    {#each tags as tag (tag)}
      <Button
        variant={language === tag ? 'secondary' : 'ghost'}
        size="sm"
        onclick={() => applyLanguage(tag)}>{labels[tag]}</Button
      >
    {/each}
  </div>

  <Card>
    <CardContent>
      <div class="flex flex-col gap-3 text-sm">
        <h3 class="font-semibold">{m('whatTitle')}</h3>
        <p>{m('what')}</p>

        <h3 class="mt-2 font-semibold">{m('flowTitle')}</h3>
        <ol class="ml-4 list-decimal">
          <li>{m('flow1')}</li>
          <li>{m('flow2')}</li>
          <li>{m('flow3')}</li>
          <li>{m('flow4')}</li>
          <li>{m('flow5')}</li>
        </ol>

        <h3 class="mt-2 font-semibold">{m('linuxTitle')}</h3>
        <p>{m('linux')}</p>
        <pre class="bg-neutral-100 rounded overflow-x-auto p-2 text-xs">{m(
            'linuxSample',
          )}</pre>
        <p>{m('linuxPath')}</p>

        <h3 class="mt-2 font-semibold">{m('localTitle')}</h3>
        <p>{m('local')}</p>

        <h3 class="mt-2 font-semibold">{m('pathTitle')}</h3>
        <p>{m('path')}</p>

        <h3 class="mt-2 font-semibold">{m('smbTitle')}</h3>
        <p>{m('smb')}</p>

        <h3 class="mt-2 font-semibold">{m('otherTitle')}</h3>
        <p>{m('other')}</p>

        <h3 class="mt-2 font-semibold">
          {m('settingsTitle')}
        </h3>
        <ul class="ml-4 list-disc">
          <li>{m('settingFolder')}</li>
          <li>{m('settingInterval')}</li>
          <li>{m('settingStorage')}</li>
          <li>{m('settingSize')}</li>
          <li>{m('settingMove')}</li>
        </ul>

        <h3 class="mt-2 font-semibold">{m('hostSettingsTitle')}</h3>
        <p>{m('hostSettings')}</p>
        <ul class="ml-4 list-disc">
          <li>{m('hostLanguage')}</li>
          <li>{m('hostMessage')}</li>
          <li>{m('hostDefaults')}</li>
        </ul>

        <figure class="mt-2">
          <img
            src={hostSettings}
            alt={m('captionSettings')}
            class="rounded border-neutral-200 border"
            loading="lazy"
          />
          <figcaption class="text-neutral-500 mt-1 text-xs">
            {m('captionSettings')}
          </figcaption>
        </figure>

        <h3 class="mt-2 font-semibold">{m('languageTitle')}</h3>
        <p>{m('language')}</p>
        <p class="text-neutral-500">{m('languageRestart')}</p>
        <p class="text-neutral-500">{m('languageContribute')}</p>
        <p>
          <a
            class="underline"
            href="https://github.com/SinglerGold2/connect-local-autostorager"
            target="_blank"
            rel="noreferrer">{m('languageRepo')}</a
          >
        </p>

        <h3 class="mt-2 font-semibold">{m('namesTitle')}</h3>
        <p>{m('names')}</p>

        <h3 class="mt-2 font-semibold">{m('failedTitle')}</h3>
        <p>{m('failed')}</p>

        <h3 class="mt-2 font-semibold">
          {m('precedenceTitle')}
        </h3>
        <p>{m('precedence')}</p>

        <h3 class="mt-2 font-semibold">{m('checkTitle')}</h3>
        <p>{m('check')}</p>

        <h3 class="mt-2 font-semibold">
          {m('picturesTitle')}
        </h3>
        <div class="flex flex-col gap-4">
          <figure>
            <img
              src={step1}
              alt={m('picture1')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {m('picture1')}
            </figcaption>
          </figure>
          <figure>
            <img
              src={step2}
              alt={m('picture2')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {m('picture2')}
            </figcaption>
          </figure>
          <figure>
            <img
              src={step3}
              alt={m('picture3')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {m('picture3')}
            </figcaption>
          </figure>
          <figure>
            <img
              src={step4}
              alt={m('picture4')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {m('picture4')}
            </figcaption>
          </figure>
          <figure>
            <img
              src={step5}
              alt={m('picture5')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {m('picture5')}
            </figcaption>
          </figure>
        </div>
      </div>
    </CardContent>
  </Card>
</div>
