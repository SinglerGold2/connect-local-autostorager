<script lang="ts">
  import { Card, CardContent, PageHeader } from '@prusa/connect-ui/primitives';

  import step1 from './assets/1-export-from-slicer.webp';
  import step2 from './assets/2-save-into-the-share.webp';
  import step3 from './assets/3-enter-the-folder.webp';
  import step4 from './assets/4-it-appears-in-storage.webp';
  import step5 from './assets/5-with-its-slicer-details.webp';

  import { heldHost } from './host';

  const t = heldHost().t;
</script>

<div class="flex flex-col gap-4">
  <PageHeader title={t('autostorager.manual.title')} />

  <Card>
    <CardContent>
      <div class="flex flex-col gap-3 text-sm">
        <h3 class="font-medium">{t('autostorager.manual.whatTitle')}</h3>
        <p>{t('autostorager.manual.what')}</p>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.flowTitle')}</h3>
        <ol class="ml-4 list-decimal">
          <li>{t('autostorager.manual.flow1')}</li>
          <li>{t('autostorager.manual.flow2')}</li>
          <li>{t('autostorager.manual.flow3')}</li>
          <li>{t('autostorager.manual.flow4')}</li>
          <li>{t('autostorager.manual.flow5')}</li>
        </ol>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.linuxTitle')}</h3>
        <p>{t('autostorager.manual.linux')}</p>
        <pre
          class="bg-neutral-100 rounded overflow-x-auto p-2 text-xs">services:
  prusa-connect:
    volumes:
      - /srv/autostorager:/incoming</pre>
        <p>{t('autostorager.manual.linuxPath')}</p>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.localTitle')}</h3>
        <p>{t('autostorager.manual.local')}</p>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.pathTitle')}</h3>
        <p>{t('autostorager.manual.path')}</p>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.smbTitle')}</h3>
        <p>{t('autostorager.manual.smb')}</p>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.otherTitle')}</h3>
        <p>{t('autostorager.manual.other')}</p>

        <h3 class="mt-2 font-medium">
          {t('autostorager.manual.settingsTitle')}
        </h3>
        <ul class="ml-4 list-disc">
          <li>{t('autostorager.manual.settingFolder')}</li>
          <li>{t('autostorager.manual.settingInterval')}</li>
          <li>{t('autostorager.manual.settingStorage')}</li>
          <li>{t('autostorager.manual.settingSize')}</li>
          <li>{t('autostorager.manual.settingMove')}</li>
        </ul>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.namesTitle')}</h3>
        <p>{t('autostorager.manual.names')}</p>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.failedTitle')}</h3>
        <p>{t('autostorager.manual.failed')}</p>

        <h3 class="mt-2 font-medium">
          {t('autostorager.manual.precedenceTitle')}
        </h3>
        <p>{t('autostorager.manual.precedence')}</p>

        <h3 class="mt-2 font-medium">{t('autostorager.manual.checkTitle')}</h3>
        <p>{t('autostorager.manual.check')}</p>

        <h3 class="mt-2 font-medium">
          {t('autostorager.manual.picturesTitle')}
        </h3>
        <div class="flex flex-col gap-4">
          <figure>
            <img
              src={step1}
              alt={t('autostorager.manual.picture1')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {t('autostorager.manual.picture1')}
            </figcaption>
          </figure>
          <figure>
            <img
              src={step2}
              alt={t('autostorager.manual.picture2')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {t('autostorager.manual.picture2')}
            </figcaption>
          </figure>
          <figure>
            <img
              src={step3}
              alt={t('autostorager.manual.picture3')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {t('autostorager.manual.picture3')}
            </figcaption>
          </figure>
          <figure>
            <img
              src={step4}
              alt={t('autostorager.manual.picture4')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {t('autostorager.manual.picture4')}
            </figcaption>
          </figure>
          <figure>
            <img
              src={step5}
              alt={t('autostorager.manual.picture5')}
              class="rounded border-neutral-200 border"
            />
            <figcaption class="text-neutral-500 mt-1 text-xs">
              {t('autostorager.manual.picture5')}
            </figcaption>
          </figure>
        </div>
      </div>
    </CardContent>
  </Card>
</div>
