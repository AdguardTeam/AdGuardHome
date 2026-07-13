import { createMemo, createEffect, createSignal, Show, onMount } from 'solid-js';
import { useNavigate, useParams, useLocation } from '@solidjs/router';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import type { IOption } from 'panel/lib/helpers/utils';
import { Textarea } from 'panel/common/controls/Textarea';
import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { PageLoader } from 'panel/common/ui/Loader';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';
import type { Client } from 'panel/initialState';
import {
    clientFormState,
    updateClientFormField,
    clearClientForm,
    saveClient,
    initClientForm,
    buildFormPayload,
    setFormErrors,
} from 'panel/stores/clientForm';
import { dashboardState, getClients } from 'panel/stores/dashboard';
import { validateUpstreams, validateCacheSize } from 'panel/helpers/validators';
import { RoutePath, Paths } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import { ClientsHeader } from './blocks/ClientsHeader';
import { Identifiers } from './blocks/Identifiers/Identifiers';

import s from './AddClient.module.pcss';

export const AddClient = () => {
    const navigate = useNavigate();
    const params = useParams<{ clientName?: string }>();
    const location = useLocation();

    const [nameError, setNameError] = createSignal<string | undefined>();
    const [cacheSizeError, setCacheSizeError] = createSignal<string | undefined>();
    const [upstreamsError, setUpstreamsError] = createSignal<string | undefined>();

    createEffect(() => {
        const formErrors = clientFormState.formErrors;
        setNameError(typeof formErrors.name === 'string' ? formErrors.name : undefined);
        setCacheSizeError(
            typeof formErrors.upstreams_cache_size === 'string'
                ? formErrors.upstreams_cache_size
                : undefined,
        );
        setUpstreamsError(
            typeof formErrors.upstreams === 'string' ? formErrors.upstreams : undefined,
        );
    });

    onMount(() => {
        getClients();
    });

    const handleNameChange = (e: Event) => {
        const value = (e.target as HTMLInputElement).value;
        updateClientFormField({ field: 'name', value });
        setNameError(undefined);
    };
    createEffect(() => {
        const searchParams = new URLSearchParams(location.search);
        const id = searchParams.get('id');

        if (!id || clientFormState.mode !== 'add') {
            return;
        }

        updateClientFormField({ field: 'ids', value: [id] });
    });

    // When on the edit route and clients are loaded, initialize the form
    // from the URL param. This handles page refreshes on the edit page.
    createEffect(() => {
        const urlClientName = params.clientName;
        const clients = dashboardState.clients || [];

        if (!urlClientName || !clients.length) {
            return;
        }

        if (clientFormState.mode !== 'add' || clientFormState.originalName) {
            return;
        }

        const decodedName = decodeURIComponent(urlClientName);
        const client = clients.find((c: Client) => c.name === decodedName);

        if (!client) {
            clearClientForm();
            navigate(Paths.Clients, { replace: true });
            return;
        }

        initClientForm(buildFormPayload(client));
    });

    const isEdit = createMemo(() => clientFormState.mode === 'edit');

    const handleCancel = () => {
        clearClientForm();
        navigate(Paths.Clients);
    };

    const handleSave = async () => {
        const err = validateUpstreams(clientFormState.upstreams);
        if (err) {
            setFormErrors({ ...clientFormState.formErrors, upstreams: err });
            return;
        }
        const result = await saveClient();
        if (result) {
            navigate(Paths.Clients);
        }
    };

    const handleUseGlobalSettings = (e: Event) => {
        updateClientFormField({
            field: 'use_global_settings',
            value: (e.target as HTMLInputElement).checked,
        });
    };

    const handleUpstreamsCacheEnabled = (e: Event) => {
        updateClientFormField({
            field: 'upstreams_cache_enabled',
            value: (e.target as HTMLInputElement).checked,
        });
    };

    const handleUpstreamsBlur = () => {
        const err = validateUpstreams(clientFormState.upstreams);
        if (err) {
            setFormErrors({ ...clientFormState.formErrors, upstreams: err });
        }
    };

    const handleUpstreamsChange = (e: Event) => {
        updateClientFormField({
            field: 'upstreams',
            value: (e.target as HTMLTextAreaElement).value,
        });
        setUpstreamsError(undefined);
    };

    const handleCacheSizeChange = (e: Event) => {
        const value = Number((e.target as HTMLInputElement).value) || 0;
        updateClientFormField({ field: 'upstreams_cache_size', value });
        setCacheSizeError(undefined);
    };

    const handleCacheSizeBlur = () => {
        if (!clientFormState.upstreams_cache_enabled) return;
        const err = validateCacheSize(clientFormState.upstreams_cache_size, true);
        setCacheSizeError(err);
        if (err) {
            setFormErrors({ ...clientFormState.formErrors, upstreams_cache_size: err });
        }
    };

    const tagOptions = createMemo(() =>
        (dashboardState.supportedTags || []).map((tag: string) => ({
            label: tag,
            value: tag,
        })),
    );

    const tagValue = createMemo(() =>
        clientFormState.tags.map((t: string) => ({ label: t, value: t })),
    );

    const handleTagChange = (selected: IOption<string> | IOption<string>[] | null) => {
        const tags = Array.isArray(selected) ? selected.map((s) => s.value) : [];
        updateClientFormField({ field: 'tags', value: tags });
    };

    const protectionRoute = createMemo(() =>
        isEdit() ? RoutePath.ClientsEditProtection : RoutePath.ClientsProtection,
    );

    const blockedServicesRoute = createMemo(() =>
        isEdit() ? RoutePath.ClientsEditBlockedServices : RoutePath.ClientsBlockedServices,
    );

    const routeProps = createMemo(() =>
        isEdit() ? { clientName: encodeURIComponent(clientFormState.originalName) } : undefined,
    );

    return (
        <div class={cn(theme.layout.container, s.containerOverride)} data-testid="client-form">
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <ClientsHeader currentTitle={intl.getMessage('clients_add')} />

                <Show when={clientFormState.processingSave}>
                    <PageLoader />
                </Show>

                <Show when={!clientFormState.processingSave}>
                    <div class={s.fieldGroupInput}>
                        <Input
                            id="client-name"
                            data-testid="client-form-name"
                            type="text"
                            value={clientFormState.name}
                            onChange={handleNameChange}
                            onInput={handleNameChange}
                            placeholder={intl.getMessage('clients_add_placeholder')}
                            label={intl.getMessage('clients_add_name')}
                            size="large"
                            error={!!nameError()}
                            errorMessage={nameError()}
                        />
                    </div>

                    <div class={s.section}>
                        <Identifiers />
                    </div>

                    <div class={s.section}>
                        <div class={cn(theme.text.t2, theme.text.semibold, s.fieldLabel)}>
                            {intl.getMessage('clients_tags')}
                        </div>
                        <div class={cn(theme.text.t3, s.fieldDesc)}>
                            {intl.getMessage('clients_tags_desc', {
                                a: (text: string) => (
                                    <a
                                        href="https://adguard-dns.io/kb/general/dns-filtering-syntax/#ctag"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                    >
                                        {text}
                                    </a>
                                ),
                            })}
                        </div>
                        <Select
                            options={tagOptions()}
                            value={tagValue()}
                            onChange={handleTagChange}
                            placeholder={intl.getMessage('clients_tags_placeholder')}
                            isMulti
                            size="responsive"
                            height="big"
                            closeMenuOnSelect={false}
                            isSearchable
                        />
                    </div>

                    <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                        {intl.getMessage('settings')}
                    </h2>

                    <SwitchGroup
                        id="use-global-settings"
                        title={intl.getMessage('clients_use_global_settings')}
                        description={intl.getMessage('clients_use_global_settings_desc')}
                        checked={clientFormState.use_global_settings}
                        onChange={handleUseGlobalSettings}
                    />

                    <Link
                        to={protectionRoute()}
                        props={routeProps()}
                        class={cn(s.navLink, {
                            [s.navLinkDisabled]: clientFormState.use_global_settings,
                        })}
                        disabled={clientFormState.use_global_settings}
                    >
                        <div class={s.navItem}>
                            <div class={s.navItemContent}>
                                <div class={cn(theme.text.t2, theme.text.semibold, s.navTitle)}>
                                    {intl.getMessage('clients_protection')}
                                </div>
                                <div class={cn(theme.text.t3, s.navDesc)}>
                                    {intl.getMessage('clients_protection_desc')}
                                </div>
                            </div>
                            <Icon icon="arrow" color="gray" />
                        </div>
                    </Link>

                    <Link
                        to={blockedServicesRoute()}
                        props={routeProps()}
                        class={cn(s.navLink, {
                            [s.navLinkDisabled]: clientFormState.use_global_settings,
                        })}
                        disabled={clientFormState.use_global_settings}
                    >
                        <div class={s.navItem}>
                            <div class={s.navItemContent}>
                                <div class={cn(theme.text.t2, theme.text.semibold, s.navTitle)}>
                                    {intl.getMessage('blocked_services')}
                                </div>
                                <div class={cn(theme.text.t3, s.navDesc)}>
                                    {intl.getMessage('blocked_services_desc')}
                                </div>
                            </div>
                            <Icon icon="arrow" color="gray" />
                        </div>
                    </Link>

                    <h2
                        class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet, {
                            [s.disabledText]: clientFormState.use_global_settings,
                        })}
                    >
                        {intl.getMessage('upstream_dns_servers_title')}
                    </h2>

                    <div class={s.section}>
                        <Textarea
                            id="client-upstreams"
                            value={clientFormState.upstreams}
                            onChange={handleUpstreamsChange}
                            onBlur={handleUpstreamsBlur}
                            placeholder={intl.getMessage('upstream_dns_placeholder')}
                            label={intl.getMessage('clients_upstreams_desc')}
                            rows={4}
                            disabled={clientFormState.use_global_settings}
                            errorMessage={upstreamsError()}
                        />
                    </div>

                    <SwitchGroup
                        id="use-dns-cache"
                        title={intl.getMessage('clients_use_dns_cache')}
                        checked={clientFormState.upstreams_cache_enabled}
                        onChange={handleUpstreamsCacheEnabled}
                        disabled={clientFormState.use_global_settings}
                    >
                        {clientFormState.upstreams_cache_enabled && (
                            <Input
                                id="dns-cache-size"
                                type="number"
                                value={String(clientFormState.upstreams_cache_size)}
                                onChange={handleCacheSizeChange}
                                onInput={handleCacheSizeChange}
                                onBlur={handleCacheSizeBlur}
                                placeholder={intl.getMessage('clients_dns_cache_size_placeholder')}
                                label={intl.getMessage('clients_dns_cache_size')}
                                size="large"
                                error={!!cacheSizeError()}
                                errorMessage={cacheSizeError()}
                            />
                        )}
                    </SwitchGroup>

                    <div class={s.actions}>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleSave}
                            disabled={clientFormState.processingSave}
                            data-testid="client-form-save"
                            class={s.actionButton}
                        >
                            {intl.getMessage('save_btn')}
                        </Button>
                        <Button
                            variant="secondary"
                            size="small"
                            onClick={handleCancel}
                            data-testid="client-form-cancel"
                            class={s.actionButton}
                        >
                            {intl.getMessage('cancel_btn')}
                        </Button>
                    </div>
                </Show>
            </div>
        </div>
    );
};
