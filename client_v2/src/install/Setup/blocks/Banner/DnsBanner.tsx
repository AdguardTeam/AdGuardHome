import { createMemo, Show } from 'solid-js';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';

import cn from 'clsx';

import { ADDRESS_IN_USE_TEXT, PORT_53_FAQ_LINK, STANDARD_DNS_PORT } from 'panel/helpers/constants';

import { validateRequiredValue, validateInstallPort } from 'panel/helpers/validators';
import { toNumber } from 'panel/helpers/form';
import styles from './styles.module.pcss';
import theme from 'panel/lib/theme';

type SelectOption = {
    value: string;
    label: string;
};

type Props = {
    class: string;
    dnsIp: () => string;
    dnsPort: () => number;
    setDnsIp: (value: string) => void;
    setDnsPort: (value: number) => void;
    dnsIpOptions: SelectOption[];
    dnsStatus?: string;
    isDnsFixAvailable: boolean;
    onAutofix: () => void;
};

export const DnsBanner = (props: Props) => {
    const portError = createMemo(() => {
        const port = props.dnsPort();
        const requiredError = validateRequiredValue(port);
        if (requiredError) return requiredError;
        const portError = validateInstallPort(port);
        if (portError) return portError;
        const isPortInUse = Boolean(
            props.dnsStatus && props.dnsStatus?.includes(ADDRESS_IN_USE_TEXT),
        );
        return isPortInUse ? intl.getMessage('port_in_use') : undefined;
    });

    return (
        <div class={props.class}>
            <div class={styles.bannerInputs}>
                <div class={styles.group}>
                    <div class={styles.bannerTitle}>
                        {intl.getMessage('setup_dns_title_banner')}
                    </div>

                    <div class={styles.form}>
                        <label class={styles.bannerLabel}>
                            {intl.getMessage('network_interface')}
                        </label>
                        <Select
                            options={props.dnsIpOptions}
                            value={props.dnsIpOptions.find(
                                (option) => option.value === props.dnsIp(),
                            )}
                            onChange={(selectedOption) =>
                                props.setDnsIp(selectedOption?.value ?? '')
                            }
                            placeholder={intl.getMessage('network_interface')}
                            size="responsive"
                            height="big"
                            id="install_dns_ip"
                            isSearchable={false}
                        />
                    </div>

                    <div class={styles.form}>
                        <label class={styles.bannerLabel}>
                            {intl.getMessage('install_settings_port')}
                        </label>
                        <Input
                            type="number"
                            id="install_dns_port"
                            value={props.dnsPort()}
                            errorMessage={portError()}
                            placeholder={STANDARD_DNS_PORT.toString()}
                            onChange={(e: Event) => {
                                const { value } = e.target as HTMLInputElement;
                                props.setDnsPort(toNumber(value));
                            }}
                            size="large"
                        />
                    </div>

                    <div>
                        <Show when={props.dnsStatus}>
                            <div class={cn(styles.setupError, styles.errorRow, styles.errorText)}>
                                {props.dnsStatus}
                                <Show when={props.isDnsFixAvailable}>
                                    <Button
                                        type="button"
                                        id="install_dns_fix"
                                        size="small"
                                        variant="primary"
                                        class={styles.inlineButton}
                                        onClick={props.onAutofix}
                                    />
                                </Show>
                            </div>
                            <Show when={props.isDnsFixAvailable}>
                                <div class={styles.mutedText}>
                                    <p class={styles.compactParagraph}>
                                        {intl.getMessage('autofix_warning_text')}
                                    </p>
                                    {intl.getMessage('autofix_warning_list', {
                                        p: (text: string) => (
                                            <p class={styles.compactParagraph}>{text}</p>
                                        ),
                                    })}
                                    <p class={styles.compactParagraph}>
                                        {intl.getMessage('autofix_warning_result')}
                                    </p>
                                </div>
                            </Show>
                        </Show>
                        <Show
                            when={
                                props.dnsPort() === STANDARD_DNS_PORT &&
                                !props.isDnsFixAvailable &&
                                props.dnsStatus?.includes(ADDRESS_IN_USE_TEXT)
                            }
                        >
                            <p>
                                {intl.getMessage('port_53_faq_link', {
                                    a: (text: string) => (
                                        <a
                                            href={PORT_53_FAQ_LINK}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            class={theme.link.link}
                                        >
                                            {text}
                                        </a>
                                    ),
                                })}
                            </p>
                        </Show>
                    </div>
                </div>
            </div>
        </div>
    );
};
