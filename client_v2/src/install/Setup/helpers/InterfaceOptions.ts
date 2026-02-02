import intl from 'panel/common/intl';

import { getInterfaceIp } from '../../../helpers/helpers';
import { ALL_INTERFACES_IP } from '../../../helpers/constants';
import type { InstallInterface } from '../../../initialState';

type SelectOption = {
    value: string;
    label: string;
    isDisabled: boolean;
};

const getInterfaceDisplayName = (iface: InstallInterface) => {
    const zoneAddr = iface?.ip_addresses?.find((addr) => typeof addr === 'string' && addr.includes('%'));
    const zone = zoneAddr?.split('%')[1];

    return zone || iface.name;
};

export const buildInterfaceOptions = (interfaces: InstallInterface[]): SelectOption[] => [
    {
        value: ALL_INTERFACES_IP,
        label: intl.getMessage('install_settings_all_interfaces'),
        isDisabled: false,
    },
    ...(Array.isArray(interfaces)
        ? interfaces
              .filter((iface) => iface?.ip_addresses?.length > 0)
              .map((iface) => {
                  const ip = getInterfaceIp(iface);
                  const displayName = getInterfaceDisplayName(iface);
                  const isUp = iface.flags?.includes('up');

                  return {
                      value: ip,
                      label: `${displayName} – ${ip}${!isUp ? ` (${intl.getMessage('down')})` : ''}`,
                      isDisabled: !isUp,
                  };
              })
        : []),
];
