// Store type exports for SolidJS stores
// Replaces Redux AppDispatch/AppThunk/AppGetState types
// In the Solid version, we don't need AppDispatch/AppThunk
// since actions are plain async functions that call setState directly.
// Components import store state and actions directly from store modules.

import type {
    DashboardData,
    SettingsData,
    EncryptionData,
    Client,
    AutoClient,
    DhcpData,
    DnsConfigData,
    FilteringData,
    QueryLogsData,
    RewritesData,
    ServicesData,
    ModalsData,
    ClientFormState,
    StatsData,
    AccessData,
    ClientsData,
    InstallData,
} from 'panel/initialState';

export type {
    DashboardData,
    SettingsData,
    EncryptionData,
    Client,
    AutoClient,
    DhcpData,
    DnsConfigData,
    FilteringData,
    QueryLogsData,
    RewritesData,
    ServicesData,
    ModalsData,
    ClientFormState,
    StatsData,
    AccessData,
    ClientsData,
    InstallData,
};
