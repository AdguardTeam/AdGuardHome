import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useForm, Controller } from 'react-hook-form';

import { Input } from '../../ui/Controls/Input';
import { Radio } from '../../ui/Controls/Radio';

export interface IpsetDefinition {
    name: string;
    type: string;
    family: string;
    timeout: number;
}

interface AutoCreateModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (definitions: IpsetDefinition[]) => void;
    initialDefinition?: IpsetDefinition | null;
    title: string;
}

const AutoCreateModal: React.FC<AutoCreateModalProps> = ({
    isOpen,
    onClose,
    onSave,
    initialDefinition,
    title,
}) => {
    const { t } = useTranslation();

    const { control, handleSubmit, reset, formState: { errors } } = useForm<IpsetDefinition>({
        defaultValues: {
            name: initialDefinition?.name || '',
            type: initialDefinition?.type || 'hash:ip',
            family: initialDefinition?.family || 'inet',
            timeout: initialDefinition?.timeout || 0,
        },
    });

    useEffect(() => {
        if (isOpen) {
            reset({
                name: initialDefinition?.name || '',
                type: initialDefinition?.type || 'hash:ip',
                family: initialDefinition?.family || 'inet',
                timeout: initialDefinition?.timeout || 0,
            });
        }
    }, [isOpen, initialDefinition, reset]);

    const onSubmit = (data: IpsetDefinition) => {
        // Split names by comma and create multiple definitions
        const names = data.name
            .split(',')
            .map(n => n.trim())
            .filter(n => n.length > 0);

        const definitions = names.map(name => ({
            name,
            type: data.type,
            family: data.family,
            timeout: data.timeout,
        }));

        onSave(definitions);
        reset();
        onClose();
    };

    const handleFormSubmit = (e: React.FormEvent) => {
        e.stopPropagation();
        handleSubmit(onSubmit)(e);
    };

    const handleClose = () => {
        reset();
        onClose();
    };

    const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
        if (e.target === e.currentTarget) {
            handleClose();
        }
    };

    const validateName = (value: string) => {
        if (!value || value.trim() === '') {
            return t('ipset_file_path_required');
        }

        // Split by comma and validate each name
        const names = value
            .split(',')
            .map(n => n.trim())
            .filter(n => n.length > 0);

        if (names.length === 0) {
            return t('ipset_file_path_required');
        }

        // Check each name individually
        for (const name of names) {
            if (!/^[A-Za-z0-9_-]+$/.test(name)) {
                return `${t('ipset_invalid_rule')}: "${name}"`;
            }
        }

        return undefined;
    };

    const validateTimeout = (value: number) => {
        if (value < 0) {
            return 'Timeout must be non-negative';
        }
        return undefined;
    };

    if (!isOpen) {
        return null;
    }

    const typeOptions = [
        { value: 'hash:ip', label: t('ipset_autocreate_type_ip') },
        { value: 'hash:net', label: t('ipset_autocreate_type_net') },
    ];

    const familyOptions = [
        { value: 'inet', label: t('ipset_autocreate_family_ipv4') },
        { value: 'inet6', label: t('ipset_autocreate_family_ipv6') },
    ];

    return (
        <>
            <div className="modal-backdrop fade show"></div>
            <div
                className="modal fade show d-block"
                tabIndex={-1}
                role="dialog"
                onClick={handleBackdropClick}
                style={{ zIndex: 1050 }}>
                <div
                    className="modal-dialog modal-dialog-centered"
                    role="document"
                    onClick={(e) => e.stopPropagation()}>
                    <div className="modal-content">
                        <div className="modal-header">
                            <h5 className="modal-title">{title}</h5>
                            <button
                                type="button"
                                className="close"
                                onClick={handleClose}>
                                <span className="sr-only">Close</span>
                            </button>
                        </div>
                        <form onSubmit={handleFormSubmit}>
                            <div className="modal-body">
                                <div className="form-group">
                                    <Controller
                                        name="name"
                                        control={control}
                                        rules={{ validate: validateName }}
                                        render={({ field, fieldState }) => (
                                            <Input
                                                {...field}
                                                label={t('ipset_autocreate_name')}
                                                desc={t('ipset_autocreate_name_desc')}
                                                placeholder="my_ipset1, my_ipset2, my_ipset3"
                                                error={fieldState.error?.message}
                                            />
                                        )}
                                    />
                                </div>

                                <div className="form-group">
                                    <label className="form__label">
                                        {t('ipset_autocreate_type')}
                                    </label>
                                    <div className="form__desc mb-2">
                                        {t('ipset_autocreate_type_desc')}
                                    </div>
                                    <Controller
                                        name="type"
                                        control={control}
                                        render={({ field }) => (
                                            <Radio
                                                name="type"
                                                value={field.value}
                                                options={typeOptions}
                                                onChange={field.onChange}
                                            />
                                        )}
                                    />
                                </div>

                                <div className="form-group">
                                    <label className="form__label">
                                        {t('ipset_autocreate_family')}
                                    </label>
                                    <div className="form__desc mb-2">
                                        {t('ipset_autocreate_family_desc')}
                                    </div>
                                    <Controller
                                        name="family"
                                        control={control}
                                        render={({ field }) => (
                                            <Radio
                                                name="family"
                                                value={field.value}
                                                options={familyOptions}
                                                onChange={field.onChange}
                                            />
                                        )}
                                    />
                                </div>

                                <div className="form-group">
                                    <Controller
                                        name="timeout"
                                        control={control}
                                        rules={{ validate: validateTimeout }}
                                        render={({ field, fieldState }) => (
                                            <Input
                                                {...field}
                                                type="number"
                                                label={t('ipset_autocreate_timeout')}
                                                desc={t('ipset_autocreate_timeout_desc')}
                                                placeholder="0"
                                                error={fieldState.error?.message}
                                                onChange={(e) => field.onChange(parseInt(e.target.value, 10) || 0)}
                                            />
                                        )}
                                    />
                                </div>
                            </div>
                            <div className="modal-footer">
                                <button
                                    type="button"
                                    className="btn btn-secondary"
                                    onClick={handleClose}>
                                    {t('cancel_btn')}
                                </button>
                                <button type="submit" className="btn btn-success">
                                    {t('save_btn')}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        </>
    );
};

export default AutoCreateModal;
