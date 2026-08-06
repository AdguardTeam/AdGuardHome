import { createMemo, Show } from 'solid-js';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import cn from 'clsx';

import { STANDARD_WEB_PORT, ADDRESS_IN_USE_TEXT } from 'panel/helpers/constants';

import { validateRequiredValue, validateInstallPort } from 'panel/helpers/validators';
import { toNumber } from 'panel/helpers/form';
import styles from './styles.module.pcss';

type SelectOption = {
    value: string;
    label: string;
};

type Props = {
    class: string;
    webIp: () => string;
    webPort: () => number;
    setWebIp: (value: string) => void;
    setWebPort: (value: number) => void;
    webIpOptions: SelectOption[];
    webStatus?: string;
    isWebFixAvailable: boolean;
    onAutofix: () => void;
};

export const WebBanner = (props: Props) => {
    const portError = createMemo(() => {
        const port = props.webPort();
        const requiredError = validateRequiredValue(port);
        if (requiredError) return requiredError;
        const portError = validateInstallPort(port);
        if (portError) return portError;
        const isPortInUse = Boolean(
            props.webStatus && props.webStatus?.includes(ADDRESS_IN_USE_TEXT),
        );
        return isPortInUse ? intl.getMessage('port_in_use') : undefined;
    });

    return (
        <div class={props.class}>
            <h3 class={styles.bannerTitle}>{intl.getMessage('setup_ui_title_banner')}</h3>
            <div class={styles.bannerInputs}>
                <div class={styles.form}>
                    <label for="install_web_ip" class={styles.bannerLabel}>
                        {intl.getMessage('network_interface')}
                    </label>
                    <Select
                        options={props.webIpOptions}
                        value={props.webIpOptions.find((option) => option.value === props.webIp())}
                        onChange={(selectedOption) => props.setWebIp(selectedOption?.value ?? '')}
                        placeholder={intl.getMessage('network_interface')}
                        size="responsive"
                        height="big"
                        id="install_web_ip"
                    />
                </div>

                <div class={styles.form}>
                    <label for="install_web_port" class={styles.bannerLabel}>
                        {intl.getMessage('install_settings_port')}
                    </label>
                    <Input
                        type="number"
                        id="install_web_port"
                        value={props.webPort()}
                        placeholder={STANDARD_WEB_PORT.toString()}
                        errorMessage={portError()}
                        onChange={(e: Event) => {
                            const { value } = e.target as HTMLInputElement;
                            props.setWebPort(toNumber(value));
                        }}
                        size="large"
                    />
                </div>

                <div>
                    <Show when={props.webStatus}>
                        <div class={cn(styles.setupError, styles.errorRow, styles.errorText)}>
                            {props.webStatus}
                            <Show when={props.isWebFixAvailable}>
                                <Button
                                    type="button"
                                    id="install_web_fix"
                                    size="small"
                                    variant="primary"
                                    class={styles.inlineButton}
                                    onClick={props.onAutofix}
                                />
                            </Show>
                        </div>
                    </Show>
                </div>
            </div>
        </div>
    );
};
