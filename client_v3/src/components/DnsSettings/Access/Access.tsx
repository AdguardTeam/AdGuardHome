import cn from 'clsx';

import { accessState, setAccessList } from 'panel/stores/access';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import { Form } from './Form';

export const Access = () => {
    const handleFormSubmit = (values: any) => {
        setAccessList(values);
    };

    return (
        <div>
            <h2
                class={cn(
                    theme.layout.subtitle,
                    theme.layout.subtitle_compact,
                    theme.title.h5,
                    theme.title.h4_tablet,
                )}
            >
                {intl.getMessage('access_settings_title')}
            </h2>

            <Form
                initialValues={accessState}
                onSubmit={handleFormSubmit}
                processingSet={accessState.processingSet}
            />
        </div>
    );
};
