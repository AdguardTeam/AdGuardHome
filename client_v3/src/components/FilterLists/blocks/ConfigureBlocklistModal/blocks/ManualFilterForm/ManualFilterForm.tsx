import { createSignal, createEffect, untrack } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { validatePath, validateRequiredValue } from 'panel/helpers/validators';
import theme from 'panel/lib/theme';

type Props = {
    class?: string;
    initialName?: string;
    initialUrl?: string;
};

export const ManualFilterForm = (props: Props) => {
    const [name, setName] = createSignal(untrack(() => props.initialName) ?? '');
    const [url, setUrl] = createSignal(untrack(() => props.initialUrl) ?? '');
    const [urlError, setUrlError] = createSignal<string | undefined>();

    createEffect(() => {
        if (props.initialName !== undefined) {
            setName(props.initialName);
        }
        if (props.initialUrl !== undefined) {
            setUrl(props.initialUrl);
        }
    });

    const handleUrlBlur = () => {
        const urlErr = validateRequiredValue(url()) || validatePath(url());
        setUrlError(urlErr || undefined);
    };

    return (
        <div class={cn(theme.form.group, props.class)}>
            <div class={theme.form.input}>
                <Input
                    type="text"
                    id="filters_name"
                    name="name"
                    label={intl.getMessage('name_label')}
                    placeholder={intl.getMessage('blocklist_placeholder_example')}
                    value={name()}
                    onChange={(e) => setName((e.target as HTMLInputElement).value)}
                />
            </div>

            <div class={theme.form.input}>
                <Input
                    type="text"
                    id="filters_url"
                    name="url"
                    label={intl.getMessage('blocklist_url_file_path')}
                    placeholder={intl.getMessage('blocklist_url_file_path')}
                    value={url()}
                    onChange={(e) => setUrl((e.target as HTMLInputElement).value)}
                    onBlur={handleUrlBlur}
                    errorMessage={urlError()}
                />
            </div>
        </div>
    );
};
