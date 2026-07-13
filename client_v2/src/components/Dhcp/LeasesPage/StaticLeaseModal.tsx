import { createSignal, createEffect, createMemo, Show } from 'solid-js';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { Dialog } from 'panel/common/ui/Dialog';
import {
    validateRequiredValue,
    validateIp,
    validateMac as validateMacFormat,
    validateHostname,
    validateIpNotDuplicate,
    validateMacNotDuplicate,
    validateIpv4InCidr,
    validateIpGateway,
} from 'panel/helpers/validators';
import { parseSubnetMask } from 'panel/helpers/helpers';
import { normalizeMac } from 'panel/helpers/form';

type LeaseData = {
    mac: string;
    ip: string;
    hostname: string;
};

type Props = {
    isOpen: boolean;
    isEdit: boolean;
    isMakeStatic?: boolean;
    initialData?: LeaseData;
    processingAdding: boolean;
    processingUpdating: boolean;
    staticLeases: LeaseData[];
    dhcpConfig?: { gatewayIp: string; subnetMask: string };
    onSubmit: (data: LeaseData) => void;
    onClose: () => void;
};

export const StaticLeaseModal = (props: Props) => {
    const [mac, setMac] = createSignal('');
    const [ip, setIp] = createSignal('');
    const [hostname, setHostname] = createSignal('');

    const [macError, setMacError] = createSignal('');
    const [ipError, setIpError] = createSignal('');
    const [hostnameError, setHostnameError] = createSignal('');

    // Reset form when initialData changes
    createEffect(() => {
        setMac(props.initialData?.mac || '');
        setIp(props.initialData?.ip || '');
        setHostname(props.initialData?.hostname || '');
    });

    const cidr = createMemo(() => {
        if (!props.dhcpConfig?.gatewayIp || !props.dhcpConfig?.subnetMask) {
            return undefined;
        }
        const prefix = parseSubnetMask(props.dhcpConfig.subnetMask);
        if (prefix === null) {
            return undefined;
        }
        return `${props.dhcpConfig.gatewayIp}/${prefix}`;
    });

    const isProcessing = createMemo(() => props.processingAdding || props.processingUpdating);

    const getTitle = () => {
        if (props.isMakeStatic) {
            return intl.getMessage('make_static');
        }
        if (props.isEdit) {
            return intl.getMessage('dhcp_edit_static_lease');
        }
        return intl.getMessage('dhcp_new_static_lease');
    };

    const submitLabel = () =>
        props.isMakeStatic ? intl.getMessage('make_static') : intl.getMessage('save');

    const validateMac = () => {
        const err =
            validateRequiredValue(mac()) ||
            validateMacFormat(mac()) ||
            validateMacNotDuplicate(
                props.staticLeases,
                props.isEdit ? props.initialData?.mac : undefined,
            )(mac());
        setMacError(err || '');
    };

    const validateHostnameField = () => {
        const err = validateHostname(hostname());
        setHostnameError(err || '');
    };

    const validateIpField = () => {
        const val = ip();
        const cidrVal = cidr();
        const gatewayIp = props.dhcpConfig?.gatewayIp;

        const err =
            validateRequiredValue(val) ||
            validateIp(val) ||
            validateIpNotDuplicate(
                props.staticLeases,
                props.isEdit ? props.initialData?.ip : undefined,
            )(val) ||
            (cidrVal && validateIpv4InCidr(val, { cidr: cidrVal })) ||
            (gatewayIp && validateIpGateway(val, { gatewayIp }));
        setIpError(err || '');
    };

    const isValid = createMemo(
        () => !macError() && !ipError() && !hostnameError() && mac() && ip(),
    );

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        validateMac();
        validateHostnameField();
        validateIpField();

        if (!isValid()) {
            return;
        }

        props.onSubmit({
            mac: normalizeMac(mac().trim()),
            ip: ip().trim(),
            hostname: hostname().trim(),
        });
    };

    return (
        <Dialog
            visible={props.isOpen}
            title={getTitle()}
            onClose={props.onClose}
            wrapClass="rc-dialog-update"
        >
            <form onSubmit={handleSubmit}>
                <Show when={props.isMakeStatic}>
                    <div class={theme.dialog.body}>{intl.getMessage('make_static_desc')}</div>
                </Show>
                <div class={theme.form.group}>
                    <div class={theme.form.input}>
                        <Input
                            value={mac()}
                            onChange={(e: Event) => setMac((e.target as HTMLInputElement).value)}
                            onBlur={() => {
                                const normalized = normalizeMac(mac().trim());
                                setMac(normalized);
                                validateMac();
                            }}
                            id="static_lease_mac"
                            label={intl.getMessage('dhcp_table_mac_address')}
                            placeholder={intl.getMessage('form_enter_mac')}
                            errorMessage={macError()}
                            disabled={props.isEdit || props.isMakeStatic}
                            size="large"
                        />
                    </div>

                    <div class={theme.form.input}>
                        <Input
                            value={hostname()}
                            onChange={(e: Event) =>
                                setHostname((e.target as HTMLInputElement).value)
                            }
                            onBlur={validateHostnameField}
                            id="static_lease_hostname"
                            label={intl.getMessage('dhcp_table_hostname')}
                            placeholder={intl.getMessage('form_enter_hostname')}
                            errorMessage={hostnameError()}
                            disabled={props.isMakeStatic}
                            size="large"
                        />
                    </div>

                    <div class={theme.form.input}>
                        <Input
                            value={ip()}
                            onChange={(e: Event) => setIp((e.target as HTMLInputElement).value)}
                            onBlur={validateIpField}
                            id="static_lease_ip"
                            label={intl.getMessage('dhcp_table_ip_address')}
                            placeholder={intl.getMessage('form_enter_ip')}
                            errorMessage={ipError()}
                            size="large"
                        />
                    </div>
                </div>
                <div class={theme.dialog.footer}>
                    <Button
                        type="submit"
                        variant="primary"
                        size="small"
                        disabled={isProcessing() || !isValid()}
                        class={theme.dialog.button}
                    >
                        {submitLabel()}
                    </Button>
                    <Button
                        variant="secondary"
                        size="small"
                        onClick={props.onClose}
                        disabled={isProcessing()}
                        class={theme.dialog.button}
                    >
                        {intl.getMessage('cancel')}
                    </Button>
                </div>
            </form>
        </Dialog>
    );
};
