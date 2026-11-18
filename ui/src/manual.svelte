<script lang="ts">
  import { Card, CardContent, PageHeader } from '@prusa/connect-ui/primitives';

  import step1 from './assets/1-export-from-slicer.webp';
  import step2 from './assets/2-save-into-the-share.webp';
  import step3 from './assets/3-enter-the-folder.webp';
  import step4 from './assets/4-it-appears-in-storage.webp';
  import step5 from './assets/5-with-its-slicer-details.webp';

  import { Button } from '@prusa/connect-ui/primitives';

  import { heldHost } from './host';
  import german from './manual-de.json';

  const host = heldHost();
  const t = host.t;

  // The manual is prose, so it carries its own two languages instead of
  // depending on the interface language of the whole app.
  let language = $state<'en' | 'de'>('en');

  type GermanKey = keyof typeof german;

  function m(key: string): string {
    if (language === 'de' && key in german) {
      return german[key as GermanKey];
    }

    return t('autostorager.manual.' + key);
  }
</script>

<div class="flex flex-col gap-4">
  <PageHeader title={m('title')} />

  <div class="flex items-center gap-2">
    <Button
      variant={language === 'en' ? 'secondary' : 'ghost'}
      size="sm"
      onclick={() => (language = 'en')}
    >
      {t('autostorager.page.langEn')}
    </Button>
    <Button
      variant={language === 'de' ? 'secondary' : 'ghost'}
      size="sm"
      onclick={() => (language = 'de')}
    >
      {t('autostorager.page.langDe')}
    </Button>
  </div>

  <Card>
    <CardContent>
      <div class="flex flex-col gap-3 text-sm">
        <h3 class="font-medium">{m('whatTitle')}</h3>
        <p>{m('what')}</p>

        <h3 class="mt-2 font-medium">{m('flowTitle')}</h3>
        <ol class="ml-4 list-decimal">
          <li>{m('flow1')}</li>
          <li>{m('flow2')}</li>
          <li>{m('flow3')}</li>
          <li>{m('flow4')}</li>
          <li>{m('flow5')}</li>
        </ol>

        <h3 class="mt-2 font-medium">{m('linuxTitle')}</h3>
        <p>{m('linux')}</p>
        <pre class="bg-neutral-100 rounded overflow-x-auto p-2 text-xs">{t(
            'autostorager.manual.linuxSample',
          )}</pre>
        <p>{m('linuxPath')}</p>

        <h3 class="mt-2 font-medium">{m('localTitle')}</h3>
        <p>{m('local')}</p>

        <h3 class="mt-2 font-medium">{m('pathTitle')}</h3>
        <p>{m('path')}</p>

        <h3 class="mt-2 font-medium">{m('smbTitle')}</h3>
        <p>{m('smb')}</p>

        <h3 class="mt-2 font-medium">{m('otherTitle')}</h3>
        <p>{m('other')}</p>

        <h3 class="mt-2 font-medium">
          {m('settingsTitle')}
        </h3>
        <ul class="ml-4 list-disc">
          <li>{m('settingFolder')}</li>
          <li>{m('settingInterval')}</li>
          <li>{m('settingStorage')}</li>
          <li>{m('settingSize')}</li>
          <li>{m('settingMove')}</li>
        </ul>

        <h3 class="mt-2 font-medium">{m('namesTitle')}</h3>
        <p>{m('names')}</p>

        <h3 class="mt-2 font-medium">{m('failedTitle')}</h3>
        <p>{m('failed')}</p>

        <h3 class="mt-2 font-medium">
          {m('precedenceTitle')}
        </h3>
        <p>{m('precedence')}</p>

        <h3 class="mt-2 font-medium">{m('checkTitle')}</h3>
        <p>{m('check')}</p>

        <h3 class="mt-2 font-medium">
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
