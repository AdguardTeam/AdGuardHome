import { createSignal, createMemo, Show } from 'solid-js';
import cn from 'clsx';

import {
    dnsConfigState,
    clearDnsCache,
    toggleCacheEnabled,
    toggleOptimisticCaching,
} from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { SettingRow } from 'panel/common/ui/SettingRow';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { useDialog } from 'panel/hooks/useDialog';
import { getCacheSizeSummary, getTtlSummary } from '../helpers';
import theme from 'panel/lib/theme';

import s from './Cache.module.pcss';
import { CacheInputDialog } from './blocks/CacheInputDialog';

export const Cache = () => {
    const [showClearConfirm, setShowClearConfirm] = createSignal(false);

    const cacheSizeDialog = useDialog();
    const minTtlDialog = useDialog();
    const maxTtlDialog = useDialog();

    const cacheSizeValue = createMemo(() => getCacheSizeSummary(dnsConfigState.cache_size));
    const minTtlValue = createMemo(() => getTtlSummary(dnsConfigState.cache_ttl_min));
    const maxTtlValue = createMemo(() => getTtlSummary(dnsConfigState.cache_ttl_max));

    const processing = () => dnsConfigState.processingSetConfig;

    return (
        <div class={s.section}>
            <SettingRow
                variant="switch"
                id="cache_enabled"
                title={intl.getMessage('dns_cache_title')}
                titleClass={cn(theme.title.h5, theme.title.h4_tablet, theme.text.bold, s.title)}
                descriptionClass={s.description}
                align="center"
                description={intl.getMessage('dns_cache_desc')}
                checked={!!dnsConfigState.cache_enabled}
                onChange={() => toggleCacheEnabled()}
            />

            <SettingRow
                variant="link"
                id="cache_size"
                title={intl.getMessage('dns_cache_size')}
                description={intl.getMessage('dns_cache_size_desc')}
                value={cacheSizeValue()}
                disabled={!dnsConfigState.cache_enabled}
                onClick={cacheSizeDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="override_min_ttl"
                title={intl.getMessage('dns_override_min_ttl')}
                description={intl.getMessage('dns_override_min_ttl_desc')}
                value={minTtlValue()}
                disabled={!dnsConfigState.cache_enabled}
                onClick={minTtlDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="override_max_ttl"
                title={intl.getMessage('dns_override_max_ttl')}
                description={intl.getMessage('dns_override_max_ttl_desc')}
                value={maxTtlValue()}
                disabled={!dnsConfigState.cache_enabled}
                onClick={maxTtlDialog.openDialog}
            />

            <SettingRow
                variant="switch"
                id="optimistic_caching"
                title={intl.getMessage('dns_optimistic_caching')}
                description={intl.getMessage('dns_optimistic_caching_desc')}
                checked={!!dnsConfigState.cache_optimistic}
                disabled={!dnsConfigState.cache_enabled}
                onChange={() => toggleOptimisticCaching()}
            />

            <div class={theme.form.actionRow}>
                <Button
                    variant="secondary-danger"
                    onClick={() => setShowClearConfirm(true)}
                    class={theme.form.actionButton}
                    compact
                >
                    {intl.getMessage('dns_clear_cache')}
                </Button>
            </div>

            <Show when={showClearConfirm()}>
                <ConfirmDialog
                    title={intl.getMessage('dns_clear_cache_title')}
                    text={intl.getMessage('dns_clear_cache_desc')}
                    buttonText={intl.getMessage('dns_clear_cache_confirm')}
                    cancelText={intl.getMessage('cancel')}
                    buttonVariant="danger"
                    onClose={() => setShowClearConfirm(false)}
                    onConfirm={() => {
                        clearDnsCache();
                        setShowClearConfirm(false);
                    }}
                />
            </Show>

            <CacheInputDialog
                configKey="cache_size"
                open={cacheSizeDialog.open}
                onClose={cacheSizeDialog.closeDialog}
                processing={processing()}
            />

            <CacheInputDialog
                configKey="cache_ttl_min"
                open={minTtlDialog.open}
                onClose={minTtlDialog.closeDialog}
                processing={processing()}
            />

            <CacheInputDialog
                configKey="cache_ttl_max"
                open={maxTtlDialog.open}
                onClose={maxTtlDialog.closeDialog}
                processing={processing()}
            />
        </div>
    );
};
