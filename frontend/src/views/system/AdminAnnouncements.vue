<template>
  <div class="announcement-admin">
    <div v-if="error" role="alert" class="announcement-admin__error">
      <span>{{ error }}</span>
      <button type="button" class="ann-btn ann-btn--outline ann-btn--sm" :disabled="loading" @click="load('preserve')">
        {{ t('announcements.admin.retry') }}
      </button>
    </div>

    <!-- Yayındaki duyurular -->
    <section aria-labelledby="active-announcement-title" class="announcement-admin__card">
      <div class="announcement-admin__card-header">
        <div>
          <h2 id="active-announcement-title" class="announcement-admin__card-title">
            <LucideIcon name="megaphone" class="announcement-admin__title-icon" />
            {{ t('announcements.admin.activeTitle') }}
            <span class="ann-badge ann-badge--secondary announcement-admin__count">{{ activeAnnouncements.length }}</span>
          </h2>
          <p class="announcement-admin__card-description">{{ t('announcements.admin.activeDescription') }}</p>
        </div>
        <button type="button" class="ann-btn ann-btn--outline ann-btn--sm" :disabled="loading || pending !== null"
          @click="load('preserve')">
          <LucideIcon name="refresh-cw" :size="14" />
          {{ t('announcements.admin.refresh') }}
        </button>
      </div>
      <p v-if="loading && !activeAnnouncements.length" class="announcement-admin__empty">
        {{ t('announcements.admin.loading') }}
      </p>
      <template v-else-if="activeAnnouncements.length">
        <div :key="currentActivePage" class="ann-table-wrap announcement-admin__scroll">
          <table class="ann-table">
            <thead>
              <tr>
                <th>{{ t('announcements.admin.titleLabel') }}</th>
                <th>{{ t('announcements.admin.typeLabel') }}</th>
                <th>{{ t('announcements.admin.publishedByLabel') }}</th>
                <th>{{ t('announcements.admin.publishedAtLabel') }}</th>
                <th class="announcement-admin__th-end">{{ t('announcements.admin.draftActions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in activePageItems" :key="item.id">
                <td class="announcement-admin__title-cell">
                  <p class="announcement-admin__row-title">{{ item.title }}</p>
                  <p class="announcement-admin__row-excerpt">{{ announcementPlainText(item.body) }}</p>
                </td>
                <td>{{ announcementTypeLabel(item.type, t) }}</td>
                <td>
                  <template v-if="!item.publishedByName && !item.publishedByEmail">—</template>
                  <template v-else>
                    <span v-if="item.publishedByName">{{ item.publishedByName }}</span>
                    <span v-if="item.publishedByEmail" class="announcement-admin__publisher-email"
                      :class="{ 'has-name': item.publishedByName }">
                      {{ item.publishedByName ? `(${item.publishedByEmail})` : item.publishedByEmail }}
                    </span>
                  </template>
                </td>
                <td class="announcement-admin__muted">{{ formatDate(item.publishedAt) }}</td>
                <td>
                  <div class="announcement-admin__row-actions">
                    <button type="button" class="ann-btn ann-btn--ghost ann-btn--xs ann-btn--destructive-text"
                      :disabled="pending !== null" @click="unpublishTarget = item">
                      {{ t('announcements.admin.unpublishAction') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="announcement-admin__pagination">
          <NumericPagination :page="currentActivePage" :page-count="activePageCount"
            :label="t('announcements.admin.activeTitle')" :disabled="loading || pending !== null"
            @page="activePage = $event" />
        </div>
      </template>
      <p v-else class="announcement-admin__empty">{{ t('announcements.admin.noActiveDescription') }}</p>
    </section>

    <div class="announcement-admin__workspace">
      <!-- Düzenleyici -->
      <form class="announcement-admin__card announcement-admin__editor" @submit.prevent="saveDraft">
        <div class="announcement-admin__editor-header">
          <div class="announcement-admin__editor-heading">
            <h2 class="announcement-admin__card-title">
              <LucideIcon name="file-text" class="announcement-admin__title-icon" />
              {{ t('announcements.admin.editorTitle') }}
            </h2>
            <div class="announcement-admin__editor-state">
              <button v-if="draft" type="button" class="ann-btn ann-btn--outline ann-btn--sm" :disabled="pending !== null"
                @click="newAnnouncement">
                {{ t('announcements.admin.newAnnouncement') }}
              </button>
              <span class="ann-badge ann-badge--outline announcement-admin__state-badge">
                <LucideIcon v-if="draft && !dirty" name="circle-check" :size="12" />
                {{ t(draft && !dirty ? 'announcements.admin.savedState' : 'announcements.admin.status.draft') }}
              </span>
            </div>
          </div>
          <p class="announcement-admin__editor-description">{{ t('announcements.admin.editorDescription') }}</p>
        </div>

        <div class="announcement-admin__editor-body">
          <!-- 1. İçerik -->
          <fieldset class="announcement-admin__fieldset">
            <legend class="announcement-admin__legend">
              <span class="announcement-admin__step" aria-hidden="true">1</span>
              {{ t('announcements.admin.contentSection') }}
            </legend>
            <label for="announcement-title" class="announcement-admin__field">
              <span class="announcement-admin__field-head">
                {{ t('announcements.admin.titleLabel') }}
                <span aria-hidden="true" class="announcement-admin__counter">{{ title.length }}/120</span>
              </span>
              <input id="announcement-title" v-model="title" class="ann-input announcement-admin__title-input"
                :placeholder="t('announcements.admin.titlePlaceholder')" maxlength="120" required
                :disabled="pending !== null" />
            </label>
            <div class="announcement-admin__field">
              <span class="announcement-admin__field-head">
                <label for="announcement-body">{{ t('announcements.admin.bodyLabel') }}</label>
                <span aria-hidden="true" class="announcement-admin__counter" :class="{ 'is-invalid': messageTooLong }">
                  {{ draftInput.body.length }}/1000
                </span>
              </span>
              <div>
                <div class="announcement-admin__toolbar">
                  <span class="announcement-admin__toolbar-label">{{ t('announcements.admin.formatting') }}</span>
                  <button v-for="control in formattingControls" :key="control.icon" type="button"
                    class="ann-btn ann-btn--ghost ann-btn--xs announcement-admin__toolbar-button" :aria-label="control.label"
                    :title="control.label" :disabled="pending !== null"
                    @click="insertMarkdown(control.prefix, control.suffix, control.fallback)">
                    <LucideIcon :name="control.icon" :size="14" />
                    <span class="announcement-admin__toolbar-text">{{ control.label }}</span>
                  </button>
                  <span class="announcement-admin__toolbar-hint">{{ t('announcements.admin.formattingHint') }}</span>
                </div>
                <textarea id="announcement-body" ref="bodyTextarea" v-model="body"
                  class="ann-textarea announcement-admin__body-input" :placeholder="t('announcements.admin.bodyPlaceholder')"
                  maxlength="1000" rows="5" required :disabled="pending !== null" />
              </div>
            </div>
            <div class="announcement-admin__button-box">
              <label for="announcement-button-label" class="announcement-admin__field">
                {{ t('announcements.admin.buttonTextLabel') }}
                <input id="announcement-button-label" v-model="buttonLabel" class="ann-input"
                  :placeholder="t('announcements.admin.buttonTextPlaceholder')" :disabled="pending !== null"
                  :aria-invalid="buttonInvalid" />
              </label>
              <label for="announcement-button-url" class="announcement-admin__field">
                {{ t('announcements.admin.buttonUrlLabel') }}
                <input id="announcement-button-url" v-model="buttonUrl" type="url" class="ann-input"
                  placeholder="https://example.com" :disabled="pending !== null" :aria-invalid="buttonInvalid" />
              </label>
              <p class="announcement-admin__hint announcement-admin__span-2">{{ t('announcements.admin.buttonHint') }}</p>
              <p v-if="buttonInvalid" role="alert" class="announcement-admin__field-error announcement-admin__span-2">
                {{ t('announcements.admin.buttonError') }}
              </p>
              <p v-if="messageTooLong" role="alert" class="announcement-admin__field-error announcement-admin__span-2">
                {{ t('announcements.admin.messageTooLong') }}
              </p>
            </div>
          </fieldset>

          <!-- 2. Tür ve hedef kitle -->
          <fieldset class="announcement-admin__fieldset announcement-admin__fieldset--divided">
            <legend class="announcement-admin__legend">
              <span class="announcement-admin__step" aria-hidden="true">2</span>
              {{ t('announcements.admin.targetingSection') }}
            </legend>
            <div>
              <AnnouncementTargeting :types="data.types ?? []" :type-id="typeId" :audience="audience"
                :disabled="loading || pending !== null" @type-change="typeId = $event" @audience-change="audience = $event"
                @type-created="onTypeCreated" />
              <label class="announcement-admin__banner-toggle">
                <span class="ann-checkbox">
                  <input v-model="showBanner" type="checkbox" :disabled="loading || pending !== null"
                    aria-describedby="announcement-banner-hint" />
                  <LucideIcon name="check" :size="14" />
                </span>
                <span class="announcement-admin__banner-toggle-text">
                  <span class="announcement-admin__banner-toggle-title">{{ t('announcements.admin.showBanner') }}</span>
                  <span id="announcement-banner-hint" class="announcement-admin__banner-toggle-hint">
                    {{ t('announcements.admin.showBannerHint') }}
                  </span>
                </span>
              </label>
              <div v-if="showBanner" class="announcement-admin__banner-color">
                <label for="announcement-banner-color" class="announcement-admin__label">
                  {{ t('announcements.admin.bannerColor') }}
                </label>
                <t-select id="announcement-banner-color" :value="bannerColor" :disabled="loading || pending !== null"
                  :popup-props="{ overlayClassName: 'announcement-select-popup' }"
                  @change="bannerColor = $event as AnnouncementBannerColor">
                  <template #prefixIcon>
                    <span class="announcement-admin__swatch" :class="`is-${bannerColor}`" aria-hidden="true" />
                  </template>
                  <t-option v-for="color in ANNOUNCEMENT_BANNER_COLORS" :key="color" :value="color"
                    :label="t(`announcements.admin.bannerColors.${color}`)">
                    <span class="announcement-admin__swatch-option">
                      <span class="announcement-admin__swatch" :class="`is-${color}`" aria-hidden="true" />
                      {{ t(`announcements.admin.bannerColors.${color}`) }}
                    </span>
                  </t-option>
                </t-select>
                <p class="announcement-admin__hint">{{ t('announcements.admin.bannerColorHint') }}</p>
              </div>
            </div>
          </fieldset>

          <!-- 3. Zamanlama -->
          <fieldset class="announcement-admin__fieldset announcement-admin__fieldset--divided">
            <legend class="announcement-admin__legend">
              <span class="announcement-admin__step" aria-hidden="true">3</span>
              {{ t('announcements.admin.dateSection') }}
            </legend>
            <div class="announcement-admin__dates">
              <div class="announcement-admin__field">
                <label for="announcement-start">{{ t('announcements.admin.startsAt') }}</label>
                <AnnouncementDateInput id="announcement-start" v-model:value="startsAt" :disabled="pending !== null" />
              </div>
              <div class="announcement-admin__field">
                <label for="announcement-end">{{ t('announcements.admin.endsAt') }}</label>
                <AnnouncementDateInput id="announcement-end" v-model:value="endsAt" :invalid="datesInvalid"
                  :disabled="pending !== null" />
              </div>
            </div>
            <p v-if="datesInvalid" id="announcement-date-error" role="alert" class="announcement-admin__field-error">
              {{ t('announcements.admin.dateOrderError') }}
            </p>
            <p class="announcement-admin__info">
              <LucideIcon name="info" :size="14" />
              {{ t('announcements.admin.publishImmediateHint') }}
            </p>
          </fieldset>
        </div>

        <div class="announcement-admin__editor-footer">
          <div class="announcement-admin__actions">
            <button type="button" class="ann-btn ann-btn--dark" :disabled="publishDisabled" @click="publishOpen = true">
              <LucideIcon name="send" />
              {{ t('announcements.admin.publishAction') }}
            </button>
            <button type="submit" class="ann-btn ann-btn--outline" :disabled="saveDisabled">
              <LucideIcon name="save" />
              {{ pending === 'save' ? t('announcements.admin.saving') : t('announcements.admin.saveDraft') }}
            </button>
            <button type="button" class="ann-btn ann-btn--outline" :disabled="pending !== null || testDisabledReason !== null"
              :aria-describedby="testDisabledReason ? 'announcement-test-hint' : undefined" @click="sendTest">
              <LucideIcon name="bell" />
              {{ pending === 'test' ? t('announcements.admin.sendingTest') : t('announcements.admin.testBrowser') }}
            </button>
          </div>
          <p v-if="testDisabledReason" id="announcement-test-hint" class="announcement-admin__footer-hint">
            {{ testDisabledReason }}
          </p>
          <p v-if="!draft" class="announcement-admin__footer-hint">{{ t('announcements.admin.publishWithoutSaving') }}</p>
        </div>
      </form>

      <aside class="announcement-admin__aside">
        <!-- Önizleme -->
        <section aria-labelledby="preview-title" class="announcement-admin__card announcement-admin__side-card">
          <h2 id="preview-title" class="announcement-admin__card-title">
            <LucideIcon name="eye" class="announcement-admin__title-icon" />
            {{ t('announcements.admin.previewTitle') }}
          </h2>
          <p class="announcement-admin__side-description">{{ t('announcements.admin.previewDescription') }}</p>
          <div class="announcement-admin__preview">
            <p class="announcement-admin__preview-label">{{ t('announcements.admin.inAppPreview') }}</p>
            <p class="announcement-admin__preview-hint">
              {{ t(showBanner ? 'announcements.admin.bannerPreviewHint' : 'announcements.admin.listPreviewHint') }}
            </p>
            <div v-if="showBanner" inert class="announcement-admin__preview-inert">
              <AnnouncementBannerCard :title="draftInput.title || t('announcements.admin.previewTitlePlaceholder')"
                :body="draftInput.body || t('announcements.admin.previewBodyPlaceholder')" :type="selectedType"
                :banner-color="bannerColor" dismissible />
            </div>
            <div v-else class="announcement-admin__list-preview">
              <p class="announcement-admin__list-preview-type">{{ announcementTypeLabel(selectedType, t) }}</p>
              <p class="announcement-admin__list-preview-title">
                {{ draftInput.title || t('announcements.admin.previewTitlePlaceholder') }}
              </p>
              <AnnouncementRichText class="announcement-admin__list-preview-body"
                :body="draftInput.body || t('announcements.admin.previewBodyPlaceholder')" />
            </div>
          </div>
        </section>

        <!-- Hedef kitle -->
        <section aria-labelledby="audience-title" class="announcement-admin__card announcement-admin__side-card">
          <h2 id="audience-title" class="announcement-admin__card-title">
            <LucideIcon name="users" class="announcement-admin__title-icon" />
            {{ t('announcements.admin.audienceTitle') }}
          </h2>
          <p class="announcement-admin__audience-mode">{{ t(AUDIENCE_LABELS[audience.mode]) }}</p>
          <p v-if="audiencePreview?.key === audienceKey && audiencePreview.error" role="alert"
            class="announcement-admin__field-error announcement-admin__audience-error">
            {{ audiencePreview.error }}
          </p>
          <dl class="announcement-admin__audience-stats">
            <div v-for="stat in audienceStats" :key="stat.key" class="announcement-admin__audience-stat">
              <dt>{{ stat.label }}</dt>
              <dd>{{ stat.value }}</dd>
            </div>
          </dl>
          <p v-if="!data.pushConfigured" class="announcement-admin__side-note">
            {{ t('announcements.pushNotConfigured') }}
          </p>
        </section>
      </aside>
    </div>

    <!-- Taslaklar -->
    <section aria-labelledby="drafts-title" class="announcement-admin__card">
      <div class="announcement-admin__card-header">
        <div>
          <h2 id="drafts-title" class="announcement-admin__card-title">
            <LucideIcon name="file-text" class="announcement-admin__title-icon" />
            {{ t('announcements.admin.draftsTitle') }}
            <span class="ann-badge ann-badge--secondary announcement-admin__count">{{ draftItems.length }}</span>
          </h2>
          <p class="announcement-admin__card-description">{{ t('announcements.admin.draftsDescription') }}</p>
        </div>
        <button type="button" class="ann-btn ann-btn--outline ann-btn--sm" :disabled="loading || pending !== null"
          @click="load('preserve')">
          <LucideIcon name="refresh-cw" :size="14" />
          {{ t('announcements.admin.refresh') }}
        </button>
      </div>
      <div v-if="draftItems.length" :key="currentDraftPage" class="ann-table-wrap announcement-admin__scroll">
        <table class="ann-table">
          <thead>
            <tr>
              <th>{{ t('announcements.admin.titleLabel') }}</th>
              <th>{{ t('announcements.admin.typeLabel') }}</th>
              <th>{{ t('announcements.admin.draftUpdated') }}</th>
              <th class="announcement-admin__th-end">{{ t('announcements.admin.draftActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in draftPageItems" :key="item.id">
              <td class="announcement-admin__title-cell">
                <p class="announcement-admin__row-title">{{ item.title }}</p>
                <p class="announcement-admin__row-excerpt">{{ announcementPlainText(item.body) }}</p>
              </td>
              <td>{{ announcementTypeLabel(item.type, t) }}</td>
              <td class="announcement-admin__muted">{{ formatDate(item.createdAt) }}</td>
              <td>
                <div class="announcement-admin__row-actions">
                  <button type="button" class="ann-btn ann-btn--ghost ann-btn--xs" :disabled="pending !== null"
                    @click="selectDraft(item)">
                    {{ t('announcements.admin.editDraft') }}
                  </button>
                  <button type="button" class="ann-btn ann-btn--ghost ann-btn--xs" :disabled="pending !== null"
                    @click="selectDraft(item, true)">
                    {{ t('announcements.admin.useDraft') }}
                  </button>
                  <button type="button" class="ann-btn ann-btn--ghost ann-btn--xs ann-btn--destructive-text"
                    :disabled="pending !== null" @click="deleteTarget = item">
                    {{ t('announcements.admin.deleteDraftAction') }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="announcement-admin__empty">
        {{ loading ? t('announcements.admin.loading') : t('announcements.admin.draftsEmpty') }}
      </p>
      <div v-if="draftItems.length > 0" class="announcement-admin__pagination">
        <NumericPagination :page="currentDraftPage" :page-count="draftPageCount" :label="t('announcements.admin.draftsTitle')"
          :disabled="loading || pending !== null" @page="draftPage = $event" />
      </div>
    </section>

    <!-- Geçmiş -->
    <section aria-labelledby="history-title" class="announcement-admin__card">
      <div class="announcement-admin__card-header">
        <div>
          <h2 id="history-title" class="announcement-admin__card-title">
            <LucideIcon name="history" class="announcement-admin__title-icon" />
            {{ t('announcements.admin.historyTitle') }}
            <span class="ann-badge ann-badge--secondary announcement-admin__count">{{ historyItems.length }}</span>
          </h2>
          <p class="announcement-admin__card-description">{{ t('announcements.admin.historyDescription') }}</p>
        </div>
        <button type="button" class="ann-btn ann-btn--outline ann-btn--sm" :disabled="loading || pending !== null"
          @click="load('preserve')">
          <LucideIcon name="refresh-cw" :size="14" />
          {{ t('announcements.admin.refresh') }}
        </button>
      </div>
      <div v-if="historyItems.length" :key="currentHistoryPage" class="announcement-admin__history-scroll">
        <ul class="announcement-admin__history">
          <li v-for="item in historyPageItems" :key="item.id" class="announcement-admin__history-item">
            <div class="announcement-admin__history-meta">
              <span class="ann-badge" :class="item.status === 'maintenance' ? 'ann-badge--default' : 'ann-badge--outline'">
                {{ t(`announcements.admin.status.${item.status}`) }}
              </span>
              <span class="ann-badge ann-badge--outline">{{ announcementTypeLabel(item.type, t) }}</span>
              <AnnouncementViewers v-if="item.status !== 'draft'" :announcement="item" />
              <span class="announcement-admin__history-audience">
                {{ t(AUDIENCE_LABELS[item.audience?.mode ?? 'all']) }}{{ item.audience?.ids.length ? ` · ${item.audience.ids.length}` : '' }}
              </span>
              <button type="button"
                class="ann-btn ann-btn--ghost ann-btn--xs ann-btn--destructive-text announcement-admin__history-delete"
                :disabled="pending !== null" @click="deleteTarget = item">
                {{ t('announcements.admin.deleteAction') }}
              </button>
            </div>
            <div class="announcement-admin__phase">
              <div class="announcement-admin__phase-main">
                <p class="announcement-admin__phase-label">
                  {{ t(item.status === 'draft' ? 'announcements.admin.phaseDraft' : 'announcements.admin.phasePublished') }}
                </p>
                <p class="announcement-admin__phase-title">{{ item.title }}</p>
                <AnnouncementRichText class="announcement-admin__phase-body" :body="item.body" />
                <p class="announcement-admin__phase-sender">
                  <template v-if="!senderName(item) && !senderEmail(item)">—</template>
                  <template v-else>
                    <span v-if="senderName(item)">{{ senderName(item) }}</span>
                    <span v-if="senderEmail(item)" class="announcement-admin__publisher-email"
                      :class="{ 'has-name': senderName(item) }">
                      {{ senderName(item) ? `(${senderEmail(item)})` : senderEmail(item) }}
                    </span>
                  </template>
                  · {{ formatDate(item.publishedAt ?? item.createdAt) }}
                </p>
              </div>
              <div class="announcement-admin__phase-stats">
                <dl class="announcement-admin__history-stats">
                  <div v-for="stat in historyStats(item)" :key="stat.key">
                    <dt>{{ t(`announcements.admin.${stat.key}`) }}</dt>
                    <dd>{{ stat.value }}</dd>
                  </div>
                </dl>
              </div>
            </div>
          </li>
        </ul>
      </div>
      <p v-else class="announcement-admin__empty">
        {{ loading ? t('announcements.admin.loading') : t('announcements.admin.historyEmpty') }}
      </p>
      <div v-if="historyItems.length > 0" class="announcement-admin__pagination">
        <NumericPagination :page="currentHistoryPage" :page-count="historyPageCount"
          :label="t('announcements.admin.historyTitle')" :disabled="loading || pending !== null"
          @page="historyPage = $event" />
      </div>
    </section>

    <!-- Yayından kaldırma onayı -->
    <t-dialog :visible="unpublishTarget !== null" :header="false" :footer="false" :close-btn="false"
      :close-on-overlay-click="false" width="448px" attach="body" dialog-class-name="announcement-confirm-dialog"
      @close="closeUnpublish">
      <div class="announcement-confirm">
        <div class="announcement-confirm__header">
          <h2 class="announcement-confirm__title">{{ t('announcements.admin.unpublishConfirmTitle') }}</h2>
          <p class="announcement-confirm__description">
            {{ t('announcements.admin.unpublishConfirmDescription', { title: unpublishTarget?.title ?? '' }) }}
          </p>
        </div>
        <div class="announcement-confirm__footer">
          <button type="button" class="ann-btn ann-btn--outline" :disabled="pending !== null" @click="closeUnpublish">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="ann-btn announcement-confirm__destructive" :disabled="pending !== null"
            @click="unpublishAnnouncement">
            {{ t(pending === 'unpublish' ? 'announcements.admin.unpublishing' : 'announcements.admin.unpublishAction') }}
          </button>
        </div>
      </div>
    </t-dialog>

    <!-- Silme onayı -->
    <t-dialog :visible="deleteTarget !== null" :header="false" :footer="false" :close-btn="false"
      :close-on-overlay-click="false" width="448px" attach="body" dialog-class-name="announcement-confirm-dialog"
      @close="closeDelete">
      <div class="announcement-confirm">
        <div class="announcement-confirm__header">
          <h2 class="announcement-confirm__title">
            {{ t(deleteTarget?.status === 'draft' ? 'announcements.admin.deleteDraftConfirmTitle' : 'announcements.admin.deleteConfirmTitle') }}
          </h2>
          <p class="announcement-confirm__description">
            {{ t(deleteTarget?.status === 'draft' ? 'announcements.admin.deleteDraftConfirmDescription' : 'announcements.admin.deleteConfirmDescription', { title: deleteTarget?.title ?? '' }) }}
          </p>
        </div>
        <div class="announcement-confirm__footer">
          <button type="button" class="ann-btn ann-btn--outline" :disabled="pending !== null" @click="closeDelete">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="ann-btn announcement-confirm__destructive" :disabled="pending !== null"
            @click="deleteAnnouncement">
            {{ t(pending === 'delete' ? 'announcements.admin.deleting' : deleteTarget?.status === 'draft' ? 'announcements.admin.deleteDraftAction' : 'announcements.admin.deleteAction') }}
          </button>
        </div>
      </div>
    </t-dialog>

    <!-- Yayınlama onayı -->
    <t-dialog :visible="publishOpen" :header="false" :footer="false" :close-btn="false" :close-on-overlay-click="false"
      width="512px" attach="body" dialog-class-name="announcement-confirm-dialog" @close="closePublish">
      <div class="announcement-confirm">
        <div class="announcement-confirm__header">
          <div class="announcement-confirm__publish-head">
            <div class="announcement-confirm__publish-icon">
              <LucideIcon name="send" :size="20" />
            </div>
            <div class="announcement-confirm__publish-text">
              <h2 class="announcement-confirm__title">{{ t('announcements.admin.publishConfirmTitle') }}</h2>
              <p class="announcement-confirm__description">{{ t('announcements.admin.publishConfirmDescription') }}</p>
            </div>
          </div>
          <div class="announcement-confirm__summary">
            <div class="announcement-confirm__message">
              <div class="announcement-confirm__message-meta">
                <span class="ann-badge ann-badge--secondary">{{ announcementTypeLabel(selectedType, t) }}</span>
                <span>
                  {{ draftInput.startsAt == null ? t('announcements.admin.publishScheduleImmediate') : formatDate(draftInput.startsAt) }}
                </span>
              </div>
              <p class="announcement-confirm__message-title">{{ draftInput.title }}</p>
              <AnnouncementRichText class="announcement-confirm__message-body" :body="draftInput.body" />
            </div>
            <div class="announcement-confirm__details">
              <div>
                <p class="announcement-confirm__detail-label">{{ t('announcements.admin.audienceTitle') }}</p>
                <p class="announcement-confirm__detail-value">{{ t(AUDIENCE_LABELS[audience.mode]) }}</p>
                <p class="announcement-confirm__detail-note">{{ audienceText }}</p>
              </div>
              <div>
                <p class="announcement-confirm__detail-label">{{ t('announcements.admin.dateSection') }}</p>
                <p class="announcement-confirm__detail-value">{{ formatDate(draftInput.startsAt ?? null) }}</p>
                <p class="announcement-confirm__detail-note">
                  {{ t('announcements.admin.endsAt') }}: {{ formatDate(draftInput.endsAt ?? null) }}
                </p>
              </div>
            </div>
          </div>
        </div>
        <div class="announcement-confirm__footer">
          <button type="button" class="ann-btn ann-btn--outline" :disabled="pending !== null" @click="closePublish">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="ann-btn ann-btn--default" :disabled="pending !== null" @click="publish">
            {{ pending === 'publish' ? t('announcements.admin.publishing') : t('announcements.admin.publishAction') }}
          </button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  ALL_AUDIENCE,
  ANNOUNCEMENT_BANNER_COLORS,
  AUDIENCE_LABELS,
  AnnouncementRequestError,
  announcementsApi,
  announcementTypeLabel,
  type Announcement,
  type AnnouncementAudience,
  type AnnouncementBannerColor,
  type AnnouncementList,
  type AnnouncementType,
  type AudienceCounts,
  type DraftInput,
} from '@/api/announcements'
import { publishAnnouncementPreview } from '@/composables/useAnnouncements'
import AnnouncementBannerCard from '@/components/announcements/AnnouncementBannerCard.vue'
import AnnouncementDateInput from '@/components/announcements/AnnouncementDateInput.vue'
import AnnouncementRichText from '@/components/announcements/AnnouncementRichText.vue'
import AnnouncementTargeting from '@/components/announcements/AnnouncementTargeting.vue'
import AnnouncementViewers from '@/components/announcements/AnnouncementViewers.vue'
import LucideIcon from '@/components/announcements/LucideIcon.vue'
import NumericPagination from '@/components/announcements/NumericPagination.vue'
import {
  announcementPlainText,
  isAnnouncementButtonUrl,
  splitAnnouncementBody,
  withAnnouncementButton,
} from '@/components/announcements/announcementBody'
import {
  datetimeLocalValue,
  datetimeTimestamp,
  stableSubmissionKey,
  type SubmissionKey,
} from './adminAnnouncementsUtils'

type PendingAction = 'save' | 'test' | 'publish' | 'unpublish' | 'delete'

const PAGE_SIZE = 5
const EMPTY_AUDIENCE: AudienceCounts = { users: 0, subscribedUsers: 0, devices: 0 }
const emptyList = (): AnnouncementList => ({ items: [], audience: { ...EMPTY_AUDIENCE }, pushConfigured: false })

const { t, locale } = useI18n()

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message.trim() ? error.message : fallback
}

// ---------- Liste durumu ----------
const data = ref<AnnouncementList>(emptyList())
const historyPage = ref(1)
const draftPage = ref(1)
const activePage = ref(1)
const scheduleNow = ref(0)

const draftItems = computed(() => data.value.items.filter((item) => item.status === 'draft'))
const historyItems = computed(() => data.value.items.filter((item) => item.status !== 'draft'))
const historyPageCount = computed(() => Math.max(1, Math.ceil(historyItems.value.length / PAGE_SIZE)))
const currentHistoryPage = computed(() => Math.min(historyPage.value, historyPageCount.value))
const draftPageCount = computed(() => Math.max(1, Math.ceil(draftItems.value.length / PAGE_SIZE)))
const currentDraftPage = computed(() => Math.min(draftPage.value, draftPageCount.value))

const activeAnnouncements = computed(() => {
  void scheduleNow.value
  const now = Date.now()
  return data.value.items.filter((item) =>
    item.status === 'maintenance'
    && (item.startsAt == null || item.startsAt <= now)
    && (item.endsAt == null || now < item.endsAt))
})
const activePageCount = computed(() => Math.max(1, Math.ceil(activeAnnouncements.value.length / PAGE_SIZE)))
const currentActivePage = computed(() => Math.min(activePage.value, activePageCount.value))
watch(activePageCount, (count) => { activePage.value = Math.min(activePage.value, count) })

const pageSlice = (items: Announcement[], page: number) => items.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
const activePageItems = computed(() => pageSlice(activeAnnouncements.value, currentActivePage.value))
const draftPageItems = computed(() => pageSlice(draftItems.value, currentDraftPage.value))
const historyPageItems = computed(() => pageSlice(historyItems.value, currentHistoryPage.value))

// ---------- Form durumu ----------
const draft = ref<Announcement | null>(null)
const title = ref('')
const body = ref('')
const bodyTextarea = ref<HTMLTextAreaElement | null>(null)
const buttonLabel = ref('')
const buttonUrl = ref('')
const startsAt = ref('')
const endsAt = ref('')
const typeId = ref('maintenance')
const audience = ref<AnnouncementAudience>(ALL_AUDIENCE)
const showBanner = ref(false)
const bannerColor = ref<AnnouncementBannerColor>('neutral')
const audiencePreview = ref<{ key: string; counts?: AudienceCounts; error?: string } | null>(null)

const audienceKey = computed(() => JSON.stringify(audience.value))
const audienceInvalid = computed(() => audience.value.mode !== 'all' && audience.value.ids.length === 0)
const selectedCounts = computed<AudienceCounts | undefined>(() => {
  if (audience.value.mode === 'all') return data.value.audience
  if (audienceInvalid.value) return EMPTY_AUDIENCE
  return audiencePreview.value?.key === audienceKey.value ? audiencePreview.value.counts : undefined
})
const selectedType = computed<AnnouncementType>(() =>
  data.value.types?.find((item) => item.id === typeId.value) ?? {
    id: typeId.value,
    name: typeId.value,
    builtin: ['maintenance', 'info', 'update', 'warning'].includes(typeId.value),
  })

let audienceTimer: ReturnType<typeof setTimeout> | null = null
let audienceController: AbortController | null = null
watch([audienceKey, data], () => {
  if (audienceTimer) clearTimeout(audienceTimer)
  audienceController?.abort()
  audienceTimer = null
  audienceController = null
  const key = audienceKey.value
  const selection = JSON.parse(key) as AnnouncementAudience
  if (selection.mode === 'all' || !selection.ids.length) return
  const controller = new AbortController()
  audienceController = controller
  audienceTimer = setTimeout(() => {
    announcementsApi.audience(selection, controller.signal).then(
      (counts) => { if (!controller.signal.aborted) audiencePreview.value = { key, counts } },
      () => {
        if (!controller.signal.aborted) audiencePreview.value = { key, error: t('announcements.admin.targetsFailed') }
      },
    )
  }, 200)
})

const loading = ref(true)
const pending = ref<PendingAction | null>(null)
const error = ref<string | null>(null)
const publishOpen = ref(false)
const unpublishTarget = ref<Announcement | null>(null)
const deleteTarget = ref<Announcement | null>(null)
let pendingLock = false
let selectedDraftId: string | null = null
let createKey: SubmissionKey | null = null
let publishKey: SubmissionKey | null = null

function hydrateDraft(item: Announcement | null) {
  const content = splitAnnouncementBody(item?.body ?? '')
  selectedDraftId = item?.id ?? null
  title.value = item?.title ?? ''
  body.value = content.body
  buttonLabel.value = content.buttonLabel
  buttonUrl.value = content.buttonUrl
  startsAt.value = datetimeLocalValue(item?.startsAt ?? null)
  endsAt.value = datetimeLocalValue(item?.endsAt ?? null)
  typeId.value = item?.typeId ?? 'maintenance'
  audience.value = item?.audience ?? ALL_AUDIENCE
  showBanner.value = item?.showBanner ?? false
  bannerColor.value = item?.bannerColor ?? 'neutral'
}

function selectDraft(item: Announcement, preparePublish = false) {
  draft.value = item
  hydrateDraft(item)
  if (preparePublish) publishOpen.value = true
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  document.getElementById('announcement-title')?.scrollIntoView({
    behavior: (reduce ? 'instant' : 'smooth') as ScrollBehavior,
    block: 'center',
  })
}

function newAnnouncement() {
  draft.value = null
  hydrateDraft(null)
  createKey = null
  publishKey = null
  error.value = null
}

async function load(editor: 'preserve' | 'replace' | 'conflict' = 'preserve') {
  loading.value = true
  error.value = null
  try {
    const result = await announcementsApi.list()
    const nextDraft = editor === 'conflict'
      ? result.items.find((item) => item.id === selectedDraftId && item.status === 'draft') ?? null
      : result.items.find((item) => item.status === 'draft') ?? null
    data.value = { ...result, items: result.items ?? [] }
    if (editor !== 'preserve') {
      draft.value = nextDraft
      hydrateDraft(nextDraft)
    }
  } catch (loadError) {
    error.value = errorMessage(loadError, t('announcements.admin.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())

// Bekleyen teslimat varken 7 sn'de bir, zamanlanmış duyuru varken 15 sn'de bir tazele.
const hasPendingDeliveries = computed(() => data.value.items.some((item) => (item.delivery?.pending ?? 0) > 0))
let deliveryTimer: ReturnType<typeof setInterval> | null = null
watch(hasPendingDeliveries, (active) => {
  if (deliveryTimer) clearInterval(deliveryTimer)
  deliveryTimer = active ? setInterval(() => void load('preserve'), 7000) : null
}, { immediate: true })

const hasSchedule = computed(() =>
  data.value.items.some((item) => item.status === 'maintenance' && (item.startsAt != null || item.endsAt != null)))
let scheduleTimer: ReturnType<typeof setInterval> | null = null
watch(hasSchedule, (active) => {
  if (scheduleTimer) clearInterval(scheduleTimer)
  scheduleTimer = active ? setInterval(() => { scheduleNow.value = Date.now() }, 15_000) : null
}, { immediate: true })

onBeforeUnmount(() => {
  if (deliveryTimer) clearInterval(deliveryTimer)
  if (scheduleTimer) clearInterval(scheduleTimer)
  if (audienceTimer) clearTimeout(audienceTimer)
  audienceController?.abort()
})

// ---------- Türetilmiş form değerleri ----------
const draftInput = computed<DraftInput>(() => ({
  title: title.value.trim(),
  body: withAnnouncementButton(body.value, buttonLabel.value, buttonUrl.value),
  startsAt: datetimeTimestamp(startsAt.value),
  endsAt: datetimeTimestamp(endsAt.value),
  typeId: typeId.value,
  audience: audience.value,
  showBanner: showBanner.value,
  bannerColor: bannerColor.value,
}))
const buttonInvalid = computed(() => {
  const hasLabel = Boolean(buttonLabel.value.trim())
  const hasUrl = Boolean(buttonUrl.value.trim())
  return hasLabel !== hasUrl || (hasUrl && !isAnnouncementButtonUrl(buttonUrl.value.trim()))
})
const messageTooLong = computed(() => draftInput.value.body.length > 1000)
const draftFingerprint = computed(() => JSON.stringify(draftInput.value))
// Rethra'da tarayıcı anlık bildirimi (Web Push) yok; test gönderimi referanstaki gibi kurulum gerektirir.
const testDisabledReason = computed(() => {
  if (loading.value) return t('announcements.admin.loading')
  if (!data.value.pushConfigured) return t('announcements.admin.testNeedsSetup')
  return t('announcements.pushUnavailable')
})
const savedFingerprint = computed(() => {
  const item = draft.value
  return item
    ? JSON.stringify({
      title: item.title,
      body: item.body,
      startsAt: item.startsAt,
      endsAt: item.endsAt,
      typeId: item.typeId ?? 'maintenance',
      audience: item.audience ?? ALL_AUDIENCE,
      showBanner: item.showBanner ?? false,
      bannerColor: item.bannerColor ?? 'neutral',
    })
    : null
})
const dirty = computed(() => draftFingerprint.value !== savedFingerprint.value)
const datesInvalid = computed(() =>
  draftInput.value.startsAt != null && draftInput.value.endsAt != null && draftInput.value.endsAt <= draftInput.value.startsAt)
const contentInvalid = computed(() =>
  audienceInvalid.value || datesInvalid.value || buttonInvalid.value || messageTooLong.value
  || !draftInput.value.title || !draftInput.value.body)
const publishDisabled = computed(() => pending.value !== null || contentInvalid.value || !selectedCounts.value)
const saveDisabled = computed(() => pending.value !== null || contentInvalid.value || !dirty.value)

function formatDate(value: number | null) {
  return value == null
    ? t('announcements.admin.notSet')
    : new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(value)
}

const audienceText = computed(() => t('announcements.admin.audienceSummary', {
  users: selectedCounts.value?.users ?? 0,
  subscribedUsers: selectedCounts.value?.subscribedUsers ?? 0,
  devices: selectedCounts.value?.devices ?? 0,
}))

const audienceStats = computed(() => [
  { key: 'users', value: selectedCounts.value?.users ?? '—', label: t('announcements.admin.users') },
  { key: 'subscribers', value: selectedCounts.value?.subscribedUsers ?? '—', label: t('announcements.admin.subscribedUsers') },
  { key: 'devices', value: selectedCounts.value?.devices ?? '—', label: t('announcements.admin.devices') },
])

function historyStats(item: Announcement) {
  const counts = item.audienceCounts ?? EMPTY_AUDIENCE
  return [
    { key: 'targetUsers', value: counts.users },
    { key: 'viewerMetric', value: item.viewerCount ?? 0 },
    { key: 'pushSubscribers', value: counts.subscribedUsers },
    { key: 'subscribedDevices', value: counts.devices },
  ]
}

const senderName = (item: Announcement) => (item.status === 'draft' ? item.createdByName : item.publishedByName)
const senderEmail = (item: Announcement) => (item.status === 'draft' ? item.createdByEmail : item.publishedByEmail)

function onTypeCreated(type: AnnouncementType) {
  data.value = { ...data.value, types: [...(data.value.types ?? []), type] }
}

// ---------- Biçimlendirme ----------
const formattingControls = computed(() => [
  { label: t('announcements.admin.bold'), icon: 'bold', prefix: '**', suffix: '**', fallback: 'kalın metin' },
  { label: t('announcements.admin.italic'), icon: 'italic', prefix: '*', suffix: '*', fallback: 'italik metin' },
])

function insertMarkdown(prefix: string, suffix: string, fallback: string) {
  const textarea = bodyTextarea.value
  if (!textarea) return
  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const selected = textarea.value.slice(start, end) || fallback
  body.value = `${textarea.value.slice(0, start)}${prefix}${selected}${suffix}${textarea.value.slice(end)}`
  const selectionStart = start + prefix.length
  textarea.focus()
  void nextTick(() => requestAnimationFrame(() =>
    textarea.setSelectionRange(selectionStart, selectionStart + selected.length)))
}

// ---------- İşlemler ----------
async function handleFailure(cause: unknown, fallback: string) {
  if (cause instanceof AnnouncementRequestError && cause.status === 409) {
    const message = t('announcements.admin.conflict')
    await load('conflict')
    error.value = message
    void MessagePlugin.error(message)
    return
  }
  const message = errorMessage(cause, fallback)
  error.value = message
  void MessagePlugin.error(message === fallback ? fallback : `${fallback} ${message}`)
}

function begin(action: PendingAction): boolean {
  if (pendingLock) return false
  pendingLock = true
  pending.value = action
  return true
}

function finish() {
  pendingLock = false
  pending.value = null
}

function upsertItem(saved: Announcement) {
  data.value = { ...data.value, items: [saved, ...data.value.items.filter((item) => item.id !== saved.id)] }
}

async function saveDraft() {
  if (contentInvalid.value || !begin('save')) return
  error.value = null
  try {
    let saved: Announcement
    if (draft.value) {
      saved = await announcementsApi.update(draft.value.id, { ...draftInput.value, version: draft.value.version })
    } else {
      createKey = stableSubmissionKey(createKey, draftFingerprint.value)
      saved = await announcementsApi.create({ ...draftInput.value, idempotencyKey: createKey.key })
    }
    draft.value = saved
    hydrateDraft(saved)
    upsertItem(saved)
    createKey = null
    void MessagePlugin.success(t('announcements.admin.draftSaved'))
  } catch (cause) {
    await handleFailure(cause, t('announcements.admin.saveFailed'))
  } finally {
    finish()
  }
}

function sendTest() {
  if (testDisabledReason.value !== null || contentInvalid.value || !begin('test')) return
  // Sunucu anlık bildirimi desteklemediğinden yalnızca bu tarayıcıdaki uygulama içi önizleme gösterilir.
  publishAnnouncementPreview({
    title: draftInput.value.title,
    body: draftInput.value.body,
    type: selectedType.value,
    bannerColor: bannerColor.value,
    showBanner: showBanner.value,
  })
  finish()
}

async function publish() {
  if (contentInvalid.value || !selectedCounts.value || !begin('publish')) return
  error.value = null
  const input = draftInput.value
  const current = draft.value
  try {
    let saved: Announcement
    if (!current) {
      createKey = stableSubmissionKey(createKey, draftFingerprint.value)
      saved = await announcementsApi.create({ ...input, idempotencyKey: createKey.key })
    } else if (dirty.value) {
      saved = await announcementsApi.update(current.id, { ...input, version: current.version })
      draft.value = saved
      hydrateDraft(saved)
      upsertItem(saved)
    } else {
      saved = current
    }
    publishKey = stableSubmissionKey(publishKey, JSON.stringify({ id: saved.id, ...input }))
    await announcementsApi.publish(saved.id, {
      version: saved.version,
      idempotencyKey: publishKey.key,
      keepDraft: current !== null,
    })
    publishKey = null
    createKey = null
    publishOpen.value = false
    await load('preserve')
    if (!current) hydrateDraft(null)
    void MessagePlugin.success(t('announcements.admin.published'))
  } catch (cause) {
    await handleFailure(cause, t('announcements.admin.publishFailed'))
  } finally {
    finish()
  }
}

async function deleteAnnouncement() {
  const target = deleteTarget.value
  if (!target || !begin('delete')) return
  const deletingDraft = target.status === 'draft'
  error.value = null
  try {
    await announcementsApi.delete(target.id, { version: target.version })
    data.value = { ...data.value, items: data.value.items.filter((item) => item.id !== target.id) }
    if (draft.value?.id === target.id) {
      draft.value = null
      hydrateDraft(null)
      createKey = null
      publishKey = null
    }
    deleteTarget.value = null
    await load('preserve')
    void MessagePlugin.success(t(deletingDraft ? 'announcements.admin.draftDeleted' : 'announcements.admin.deleted'))
  } catch (cause) {
    deleteTarget.value = null
    await handleFailure(cause, t(deletingDraft ? 'announcements.admin.deleteDraftFailed' : 'announcements.admin.deleteFailed'))
  } finally {
    finish()
  }
}

async function unpublishAnnouncement() {
  const target = unpublishTarget.value
  if (!target || !begin('unpublish')) return
  error.value = null
  try {
    await announcementsApi.unpublish(target.id, { version: target.version })
    unpublishTarget.value = null
    await load('preserve')
    void MessagePlugin.success(t('announcements.admin.unpublished'))
  } catch (cause) {
    unpublishTarget.value = null
    await handleFailure(cause, t('announcements.admin.unpublishFailed'))
  } finally {
    finish()
  }
}

function closeUnpublish() {
  if (!pending.value) unpublishTarget.value = null
}

function closeDelete() {
  if (!pending.value) deleteTarget.value = null
}

function closePublish() {
  if (!pending.value) publishOpen.value = false
}
</script>

<style lang="less">
// Referans: rag-platform admin-announcements.tsx (Tailwind) — belirteç tabanlı karşılıkları.
.announcement-admin {
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding-bottom: 24px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  overflow-wrap: anywhere;

  p {
    margin: 0;
  }

  h2 {
    margin: 0;
  }
}

.announcement-admin__error {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid color-mix(in srgb, var(--td-error-color) 40%, transparent);
  border-radius: var(--app-radius-token-xl);
  background: color-mix(in srgb, var(--td-error-color) 5%, transparent);
  color: var(--td-error-color);
}

.announcement-admin__card {
  min-width: 0;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  border-radius: var(--app-radius-token-2xl);
  background: var(--td-bg-color-container);
}

.announcement-admin__card-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 20px;
}

.announcement-admin__card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--app-font-heading);
  font-size: var(--app-text-lg);
  font-weight: 600;
  letter-spacing: -0.025em;
  line-height: 1.5;
}

.announcement-admin__title-icon {
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
}

.announcement-admin__count {
  font-variant-numeric: tabular-nums;
  letter-spacing: 0;
}

.announcement-admin__card-description {
  color: var(--td-text-color-secondary);
  line-height: 1.43;
}

.announcement-admin__empty {
  padding: 32px 20px;
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 60%, transparent);
  color: var(--td-text-color-secondary);
}

.announcement-admin__muted {
  color: var(--td-text-color-secondary);
}

.announcement-admin__scroll {
  max-height: 32rem;
  overflow: auto;
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 60%, transparent);
}

.announcement-admin__th-end {
  text-align: right;

  .ann-table & {
    text-align: right;
  }
}

.announcement-admin__title-cell {
  min-width: 224px;
  max-width: 448px;

  .ann-table & {
    white-space: normal;
  }
}

.announcement-admin__row-title {
  font-weight: 500;
  word-break: break-word;
}

.announcement-admin__row-excerpt {
  display: -webkit-box;
  margin-top: 4px;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-admin__publisher-email {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);

  &.has-name {
    margin-left: 4px;
  }
}

.announcement-admin__row-actions {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
}

.announcement-admin__pagination {
  display: flex;
  justify-content: flex-end;
  padding: 12px 20px;
  border-top: 1px solid var(--td-component-stroke);
}

// ---------- Düzenleyici + yan panel ----------
.announcement-admin__workspace {
  display: grid;
  align-items: start;
  gap: 24px;
}

.announcement-admin__editor {
  box-shadow: 0 1px 2px 0 color-mix(in srgb, #000 5%, transparent);
}

.announcement-admin__editor-header {
  padding: 20px;
  border-bottom: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
}

.announcement-admin__editor-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.announcement-admin__editor-state {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.announcement-admin__state-badge {
  gap: 6px;
}

.announcement-admin__editor-description {
  max-width: 32rem;
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  line-height: 1.625;

  .announcement-admin & {
    margin-top: 4px;
  }
}

.announcement-admin__editor-body {
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 24px 20px;
}

.announcement-admin__fieldset {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 16px;
  margin: 0;
  padding: 0;
  border: 0;

  &--divided {
    padding-top: 24px;
    border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  }
}

.announcement-admin__legend {
  display: flex;
  float: left;
  width: 100%;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 0;
  font-size: var(--app-text-base);
  font-weight: 600;

  + * {
    clear: both;
  }
}

.announcement-admin__step {
  display: flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  border-radius: var(--app-radius-token-md);
  background: var(--app-surface-muted);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-admin__field {
  display: grid;
  gap: 6px;
  font-size: var(--app-text-base);
  font-weight: 500;
}

.announcement-admin__label {
  font-size: var(--app-text-base);
  font-weight: 500;
}

.announcement-admin__field-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.announcement-admin__counter {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 400;
  font-variant-numeric: tabular-nums;

  &.is-invalid {
    color: var(--td-error-color);
  }
}

.announcement-admin__title-input.ann-input {
  height: 44px;
  border-radius: var(--app-radius-token);
}

.announcement-admin__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border: 1px solid var(--td-component-stroke);
  border-bottom: 0;
  border-radius: var(--app-radius-token) var(--app-radius-token) 0 0;
  background: color-mix(in srgb, var(--app-surface-muted) 35%, transparent);
}

.announcement-admin__toolbar-label {
  margin-right: 4px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 500;
}

.announcement-admin__toolbar-button.ann-btn {
  gap: 6px;
  padding: 0 8px;
  border-radius: var(--app-radius-token-md);
}

.announcement-admin__toolbar-text {
  display: none;
}

.announcement-admin__toolbar-hint {
  margin-left: auto;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-micro);
  font-weight: 400;
}

.announcement-admin__body-input.ann-textarea {
  min-height: 176px;
  border-radius: 0 0 var(--app-radius-token) var(--app-radius-token);
  line-height: 1.625;
}

.announcement-admin__button-box {
  display: grid;
  gap: 12px;
  padding: 16px;
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  border-radius: var(--app-radius-token);
  background: color-mix(in srgb, var(--app-surface-muted) 20%, transparent);
}

.announcement-admin__hint {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 400;
}

.announcement-admin__field-error {
  color: var(--td-error-color);
  font-size: var(--app-text-base);
  font-weight: 400;
}

.announcement-admin__banner-toggle {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-top: 16px;
  padding: 16px;
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  border-radius: var(--app-radius-token-xl);
  background: color-mix(in srgb, var(--app-surface-muted) 25%, transparent);
  cursor: pointer;
  transition: background-color var(--app-motion-fast) ease;

  &:hover {
    background: color-mix(in srgb, var(--app-surface-muted) 50%, transparent);
  }

  .ann-checkbox {
    margin-top: 2px;
  }
}

.announcement-admin__banner-toggle-text {
  min-width: 0;
}

.announcement-admin__banner-toggle-title {
  font-size: var(--app-text-base);
  font-weight: 500;
}

.announcement-admin__banner-toggle-hint {
  display: block;
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-admin__banner-color {
  display: grid;
  gap: 6px;
  margin-top: 16px;
}

.announcement-admin__swatch {
  display: inline-block;
  width: 10px;
  height: 10px;
  flex-shrink: 0;
  border-radius: 50%;

  &.is-neutral { background: #71717a; }
  &.is-blue { background: #0ea5e9; }
  &.is-green { background: #10b981; }
  &.is-amber { background: #f59e0b; }
  &.is-red { background: #ef4444; }
  &.is-violet { background: #8b5cf6; }
}

.announcement-admin__swatch-option {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.announcement-admin__dates {
  display: grid;
  gap: 16px;
}

.announcement-admin__info {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  line-height: 1.625;

  svg {
    flex-shrink: 0;
    margin-top: 2px;
  }
}

.announcement-admin__editor-footer {
  padding: 16px 20px;
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  background: color-mix(in srgb, var(--app-surface-muted) 20%, transparent);
}

.announcement-admin__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;

  .ann-btn {
    min-height: 40px;
  }
}

.announcement-admin__footer-hint {
  margin-top: 12px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);

  .announcement-admin & {
    margin-top: 12px;
  }
}

.announcement-admin__aside {
  display: grid;
  min-width: 0;
  gap: 20px;
}

.announcement-admin__side-card {
  padding: 20px;
}

.announcement-admin__side-description {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  line-height: 1.625;

  .announcement-admin & {
    margin-top: 4px;
  }
}

.announcement-admin__preview {
  margin-top: 20px;
}

.announcement-admin__preview-label {
  margin-bottom: 6px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 500;

  .announcement-admin & {
    margin-bottom: 6px;
  }
}

.announcement-admin__preview-hint {
  margin-bottom: 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);

  .announcement-admin & {
    margin-bottom: 8px;
  }
}

.announcement-admin__preview-inert {
  pointer-events: none;
}

.announcement-admin__list-preview {
  padding: 12px;
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  border-radius: var(--app-radius-token);
  background: color-mix(in srgb, var(--app-surface-muted) 30%, transparent);
}

.announcement-admin__list-preview-type {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 500;
}

.announcement-admin__list-preview-title {
  margin-top: 4px;
  font-weight: 500;
  word-break: break-word;

  .announcement-admin & {
    margin-top: 4px;
  }
}

.announcement-admin__list-preview-body {
  display: -webkit-box;
  margin-top: 4px;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  color: var(--td-text-color-secondary);
}

.announcement-admin__audience-mode {
  margin-top: 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);

  .announcement-admin & {
    margin-top: 8px;
  }
}

.announcement-admin__audience-error {
  margin-top: 8px;

  .announcement-admin & {
    margin-top: 8px;
  }
}

.announcement-admin__audience-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin: 16px 0 0;
}

.announcement-admin__audience-stat {
  min-width: 0;
  padding: 12px 8px;
  border-radius: var(--app-radius-token-xl);
  background: color-mix(in srgb, var(--app-surface-muted) 45%, transparent);
  text-align: center;

  dt {
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-xs);
    line-height: 1.625;
    word-break: break-word;
  }

  dd {
    margin: 8px 0 0;
    font-family: var(--app-font-heading);
    font-size: var(--app-text-4xl);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.025em;
  }
}

.announcement-admin__side-note {
  margin-top: 12px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);

  .announcement-admin & {
    margin-top: 12px;
  }
}

// ---------- Geçmiş ----------
.announcement-admin__history-scroll {
  max-height: 32rem;
  overflow-y: auto;
}

.announcement-admin__history {
  margin: 0;
  padding: 0;
  list-style: none;

  > li + li {
    border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 60%, transparent);
  }
}

.announcement-admin__history-item {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
  transition: background-color var(--app-motion-fast) ease;

  &:hover {
    background: color-mix(in srgb, var(--app-surface-muted) 15%, transparent);
  }
}

.announcement-admin__history-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.announcement-admin__history-audience {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-admin__history-delete.ann-btn {
  margin-left: auto;
}

.announcement-admin__phase {
  display: grid;
  gap: 16px;
}

.announcement-admin__phase-main {
  min-width: 0;
}

.announcement-admin__phase-label {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 500;
}

.announcement-admin__phase-title {
  margin-top: 4px;
  font-weight: 500;
  word-break: break-word;

  .announcement-admin & {
    margin-top: 4px;
  }
}

.announcement-admin__phase-body {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
}

.announcement-admin__phase-sender {
  margin-top: 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);

  .announcement-admin & {
    margin-top: 8px;
  }
}

.announcement-admin__history-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 20px;
  margin: 0;
  padding: 12px;
  border-radius: var(--app-radius-token-xl);
  background: color-mix(in srgb, var(--app-surface-muted) 35%, transparent);
  font-size: var(--app-text-xs);
  font-variant-numeric: tabular-nums;
  text-align: left;

  dt {
    color: var(--td-text-color-secondary);
  }

  dd {
    margin: 4px 0 0;
    font-weight: 500;
  }
}

// ---------- Duyarlı düzen (Tailwind sm / lg ve kapsayıcı sorguları) ----------
@media (min-width: 640px) {
  .announcement-admin__card-header,
  .announcement-admin__editor-header,
  .announcement-admin__editor-body,
  .announcement-admin__editor-footer,
  .announcement-admin__pagination,
  .announcement-admin__history-item {
    padding-right: 24px;
    padding-left: 24px;
  }

  .announcement-admin__empty {
    padding-right: 24px;
    padding-left: 24px;
  }

  .announcement-admin__button-box,
  .announcement-admin__dates {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .announcement-admin__span-2 {
    grid-column: span 2;
  }

  .announcement-admin__history-stats {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .announcement-admin__phase-stats {
    align-self: center;
  }
}

@media (min-width: 1024px) {
  .announcement-admin__phase {
    grid-template-columns: minmax(0, 1fr) auto;
  }
}

@container (min-width: 30rem) {
  .announcement-admin__toolbar-text {
    display: inline;
  }
}

@container (min-width: 40rem) {
  .announcement-admin__aside {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@container (min-width: 52rem) {
  .announcement-admin__aside {
    grid-template-columns: minmax(0, 1fr);
  }
}

@container (min-width: 58rem) {
  .announcement-admin__workspace {
    grid-template-columns: minmax(0, 1fr) 20rem;
  }
}

// ---------- Onay pencereleri (AlertDialog) ----------
.t-dialog.announcement-confirm-dialog {
  max-width: calc(100vw - 32px);
  max-height: calc(100dvh - 32px);
  padding: 24px;
  overflow-y: auto;

  .t-dialog__body {
    padding: 0;
    color: var(--td-text-color-primary);
  }
}

.announcement-confirm {
  display: grid;
  gap: 24px;

  p,
  h2 {
    margin: 0;
  }
}

.announcement-confirm__header {
  display: grid;
  gap: 6px;
  text-align: left;
}

.announcement-confirm__title {
  font-family: var(--app-font-heading);
  font-size: var(--app-text-xl);
  font-weight: 500;
  line-height: 1.5;
}

.announcement-confirm__description {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-base);
  text-wrap: pretty;
}

.announcement-confirm__footer {
  display: flex;
  flex-direction: column-reverse;
  gap: 8px;
}

.announcement-confirm__destructive.ann-btn {
  background: color-mix(in srgb, var(--td-error-color) 10%, transparent);
  color: var(--td-error-color);

  &:hover {
    background: color-mix(in srgb, var(--td-error-color) 20%, transparent);
  }
}

.announcement-confirm__publish-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.announcement-confirm__publish-icon {
  display: flex;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: var(--app-radius-token-xl);
  background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
  color: var(--td-brand-color);
}

.announcement-confirm__publish-text {
  display: grid;
  gap: 4px;
}

.announcement-confirm__summary {
  display: grid;
  gap: 16px;
  margin-top: 0;
}

.announcement-confirm__message {
  padding: 16px;
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  border-radius: var(--app-radius-token-xl);
  background: color-mix(in srgb, var(--app-surface-muted) 30%, transparent);
  color: var(--td-text-color-primary);
}

.announcement-confirm__message-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
}

.announcement-confirm__message-title {
  margin-top: 12px;
  font-weight: 600;
  word-break: break-word;

  .announcement-confirm & {
    margin-top: 12px;
  }
}

.announcement-confirm__message-body {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-base);
}

.announcement-confirm__details {
  display: grid;
  gap: 12px;
  padding: 16px;
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  border-radius: var(--app-radius-token-xl);
}

.announcement-confirm__detail-label {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 500;
}

.announcement-confirm__detail-value {
  margin-top: 4px;
  font-size: var(--app-text-base);
  font-weight: 500;

  .announcement-confirm & {
    margin-top: 4px;
  }
}

.announcement-confirm__detail-note {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);

  .announcement-confirm & {
    margin-top: 4px;
  }
}

@media (min-width: 640px) {
  .announcement-confirm__footer {
    flex-direction: row;
    justify-content: flex-end;
  }

  .announcement-confirm__details {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
