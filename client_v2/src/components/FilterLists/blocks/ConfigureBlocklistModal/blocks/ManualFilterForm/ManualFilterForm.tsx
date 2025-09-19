import React from 'react';
import { Controller, useFormContext } from 'react-hook-form';

import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { validatePath, validateRequiredValue } from 'panel/helpers/validators';
import theme from 'panel/lib/theme';

export const ManualFilterForm = () => {
    const { control } = useFormContext();

    return (
        <div className={theme.form.group}>
            <div className={theme.form.input}>
                <Controller
                    name="name"
                    control={control}
                    render={({ field, fieldState }) => (
                        <Input
                            {...field}
                            type="text"
                            id="filters_name"
                            label={intl.getMessage('name_label')}
                            placeholder={intl.getMessage('blocklist_placeholder_example')}
                            errorMessage={fieldState.error?.message}
                        />
                    )}
                />
            </div>

            <div className={theme.form.input}>
                <Controller
                    name="url"
                    control={control}
                    rules={{
                        validate: { validateRequiredValue, validatePath },
                    }}
                    render={({ field, fieldState }) => (
                        <Input
                            {...field}
                            type="text"
                            id="filters_url"
                            label={intl.getMessage('blocklist_url_file_path')}
                            placeholder={intl.getMessage('blocklist_url_file_path')}
                            errorMessage={fieldState.error?.message}
                        />
                    )}
                />
            </div>
        </div>
    );
};
