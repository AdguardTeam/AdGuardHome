import { createEffect, onMount, onCleanup, Show } from 'solid-js';
import { HashRouter, Route, Navigate } from '@solidjs/router';

import { Sidebar } from 'panel/common/ui/Sidebar';
import { Icons } from 'panel/common/ui/Icons';
import { Footer } from 'panel/common/ui/Footer';
import { Header } from 'panel/common/ui/Header';
import { Banners } from 'panel/common/ui/Banners';
import { Settings } from 'panel/components/Settings';
import intl, { LocalesType } from 'panel/common/intl';
import { Encryption } from 'panel/components/Encryption';
import { Blocklists } from 'panel/components/FilterLists/Blocklists';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';

import { Allowlists } from 'panel/components/FilterLists/Allowlists';
import { DNSRewrites } from 'panel/components/FilterLists/DNSRewrites';
import { SetupGuide } from 'panel/components/SetupGuide';
import { Dashboard } from 'panel/components/Dashboard';
import { Dhcp } from 'panel/components/Dhcp';
import { LeasesPage } from 'panel/components/Dhcp/LeasesPage';
import { QueryLog } from 'panel/components/QueryLog';
import Toasts from '../Toasts';
import { THEMES } from '../../helpers/constants';
import { setHtmlLangAttr, setUITheme } from '../../helpers/helpers';
import { getDnsStatus, getTimerStatus, dashboardState } from '../../stores/dashboard';

import s from './styles.module.pcss';
import { DnsSettings } from '../DnsSettings';
import { PrivateReverse } from '../DnsSettings/PrivateReverse';
import { UserRules } from '../UserRules';
import { BlockedServices } from '../BlockedServices';
import { Clients } from '../Clients/Clients';
import { InactivitySchedule } from '../BlockedServices/InactivitySchedule';
import { AddClient } from '../Clients/AddClient';
import { Protection } from '../Clients/AddClient/blocks/Protection/Protection';
import { ClientBlockedServices } from '../Clients/AddClient/blocks/ClientBlockedServices';
import { ClientSchedule } from '../Clients/AddClient/blocks/ClientSchedule';

const SetupGuideRoute = () => <SetupGuide />;
const BlockedServicesRoute = () => <BlockedServices />;
const InactivityScheduleRoute = () => <InactivitySchedule />;
const ClientScheduleRoute = () => <ClientSchedule />;
const ClientBlockedServicesRoute = () => <ClientBlockedServices />;
const ProtectionRoute = () => <Protection />;
const AddClientRoute = () => <AddClient />;

const App = () => {
    onMount(() => {
        getDnsStatus();

        const handleVisibilityChange = () => {
            if (document.visibilityState === 'visible') {
                getTimerStatus();
            }
        };

        document.addEventListener('visibilitychange', handleVisibilityChange);

        onCleanup(() => {
            document.removeEventListener('visibilitychange', handleVisibilityChange);
        });
    });

    // React to language changes
    createEffect(() => {
        const language = dashboardState.language;
        const processing = dashboardState.processing;
        if (!processing && language) {
            intl.changeLanguage(language as LocalesType);
            setHtmlLangAttr(language);
            LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, language);
        }
    });

    // React to theme changes
    createEffect(() => {
        const theme = dashboardState.theme;

        if (!theme) return;

        if (theme !== THEMES.auto) {
            setUITheme(theme);
            return;
        }

        const colorSchemeMedia = window.matchMedia('(prefers-color-scheme: dark)');
        setUITheme(theme);

        const handleChange = (e: MediaQueryListEvent) => {
            if (e.matches) {
                setUITheme(THEMES.dark);
            } else {
                setUITheme(THEMES.light);
            }
        };

        colorSchemeMedia.addEventListener('change', handleChange);
        onCleanup(() => {
            colorSchemeMedia.removeEventListener('change', handleChange);
        });
    });

    return (
        <HashRouter
            root={(props) => (
                <>
                    <Header />

                    <Banners />

                    <div class={s.wrapper}>
                        <Sidebar />

                        <Show when={!dashboardState.processing && dashboardState.isCoreRunning}>
                            <div class={s.bodyWrapper}>{props.children}</div>
                        </Show>
                    </div>

                    <Footer />

                    <Toasts />

                    <Icons />
                </>
            )}
        >
            <Route path="/dashboard" component={Dashboard} />
            <Route path="/settings" component={Settings} />
            <Route path="/encryption" component={Encryption} />
            <Route path="/dns" component={DnsSettings} />
            <Route path="/dns/private-reverse" component={PrivateReverse} />
            <Route path="/blocklists" component={Blocklists} />
            <Route path="/allowlists" component={Allowlists} />
            <Route path="/user_rules" component={UserRules} />
            <Route path="/dns_rewrites" component={DNSRewrites} />
            <Route path="/dhcp" component={Dhcp} />
            <Route path="/dhcp/leases" component={LeasesPage} />
            <Route path="/guide" component={SetupGuideRoute} />
            <Route path="/logs" component={QueryLog} />
            <Route path="/blocked_services/schedule" component={InactivityScheduleRoute} />
            <Route path="/blocked_services" component={BlockedServicesRoute} />
            <Route path="/clients/add/blocked_services/schedule" component={ClientScheduleRoute} />
            <Route path="/clients/add/blocked_services" component={ClientBlockedServicesRoute} />
            <Route path="/clients/add/protection" component={ProtectionRoute} />
            <Route path="/clients/add" component={AddClientRoute} />
            <Route
                path="/clients/edit/:clientName/blocked_services/schedule"
                component={ClientScheduleRoute}
            />
            <Route
                path="/clients/edit/:clientName/blocked_services"
                component={ClientBlockedServicesRoute}
            />
            <Route path="/clients/edit/:clientName/protection" component={ProtectionRoute} />
            <Route path="/clients/edit/:clientName" component={AddClientRoute} />
            <Route path="/clients" component={Clients} />
            <Route path="/" component={() => <Navigate href="/dashboard" />} />
        </HashRouter>
    );
};

export default App;
