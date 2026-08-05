import intl from 'panel/common/intl';

import { ALL_INTERFACES_IP } from 'panel/helpers/constants';
import type { InstallInterface } from 'panel/initialState';

type SelectOption = {
    value: string;
    label: string;
};

const getInterfaceDisplayName = (iface: InstallInterface) => {
    const zoneAddr = iface?.ip_addresses?.find(
        (addr) => typeof addr === 'string' && addr.includes('%'),
    );
    const zone = zoneAddr?.split('%')[1];

    return zone || iface.name;
};

export const buildInterfaceOptions = (interfaces: InstallInterface[]): SelectOption[] => [
    {
        value: ALL_INTERFACES_IP,
        label: intl.getMessage('install_settings_all_interfaces'),
    },
    ...(Array.isArray(interfaces)
        ? interfaces
              .filter((iface) => {
                  if (!iface?.ip_addresses?.length) {
                      return false;
                  }
                  const isUp = iface.flags?.includes('up');
                  return isUp;
              })
              .flatMap((iface) => {
                  const displayName = getInterfaceDisplayName(iface);

                  return iface.ip_addresses.map((ip) => ({
                      value: ip,
                      label: `${displayName} – ${ip}`,
                  }));
              })
        : []),
];
