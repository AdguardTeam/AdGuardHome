import cn from 'clsx';

import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import theme from 'panel/lib/theme';

type Props = {
    class?: string;
    name: string;
    url: string;
    nameError?: string;
    urlError?: string;
    onNameChange: (value: string) => void;
    onUrlChange: (value: string) => void;
    onNameBlur?: () => void;
    onUrlBlur?: () => void;
};

export const ManualFilterForm = (props: Props) => (
    <div class={cn(theme.form.group, props.class)}>
        <div class={theme.form.input}>
            <Input
                type="text"
                id="filters_name"
                label={intl.getMessage('name_label')}
                placeholder={intl.getMessage('blocklist_placeholder_example')}
                value={props.name}
                onChange={(e) => props.onNameChange((e.target as HTMLInputElement).value)}
                onBlur={() => props.onNameBlur?.()}
                errorMessage={props.nameError}
                size="large"
                onCard
            />
        </div>

        <div class={theme.form.input}>
            <Input
                type="text"
                id="filters_url"
                label={intl.getMessage('blocklist_url_file_path')}
                placeholder={intl.getMessage('blocklist_url_file_path')}
                value={props.url}
                onChange={(e) => props.onUrlChange((e.target as HTMLInputElement).value)}
                onBlur={() => props.onUrlBlur?.()}
                errorMessage={props.urlError}
                size="large"
                onCard
            />
        </div>
    </div>
);
