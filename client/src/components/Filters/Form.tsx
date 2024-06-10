import React from 'react';

import { Field, reduxForm } from 'redux-form';
import { withTranslation } from 'react-i18next';
import flow from 'lodash/flow';
import classNames from 'classnames';
import { validatePath, validateRequiredValue } from '../../helpers/validators';

import { CheckboxField, renderInputField } from '../../helpers/form';
import { MODAL_OPEN_TIMEOUT, MODAL_TYPE, FORM_NAME } from '../../helpers/constants';
import filtersCatalog from '../../helpers/filters/filters';

const getIconsData = (homepage: any, source: any) => [
    {
        iconName: 'dashboard',
        href: homepage,
        className: 'ml-1',
    },
    {
        iconName: 'info',
        href: source,
    },
];

const renderIcons = (iconsData: any) =>
    iconsData.map(({ iconName, href, className = '' }: any) => (
        <a
            key={iconName}
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className={classNames('d-flex align-items-center', className)}>
            <svg className="icon icon--15 mr-1 icon--gray">
                <use xlinkHref={`#${iconName}`} />
            </svg>
        </a>
    ));

interface renderCheckboxFieldProps {
    // https://redux-form.com/8.3.0/docs/api/field.md/#props
    input: {
        name: string;
        value: string;
        checked: boolean;
        onChange: (...args: unknown[]) => unknown;
    };
    disabled: boolean;
}

const renderCheckboxField = (props: renderCheckboxFieldProps) => (
    <CheckboxField
        {...props}
        meta={{ touched: false, error: null }}
        input={{
            ...props.input,
            checked: props.disabled || props.input.checked,
        }}
    />
);

const renderFilters = ({ categories, filters }: any, selectedSources: any, t: any) =>
    Object.keys(categories).map((categoryId) => {
        const category = categories[categoryId];
        const categoryFilters: any = [];
        Object.keys(filters)
            .sort()
            .forEach((key) => {
                const filter = filters[key];
                filter.id = key;
                if (filter.categoryId === categoryId) {
                    categoryFilters.push(filter);
                }
            });

        return (
            <div key={category.name} className="modal-body__item">
                <h6 className="font-weight-bold mb-1">{t(category.name)}</h6>

                <p className="mb-3">{t(category.description)}</p>

                {categoryFilters.map((filter) => {
                    const { homepage, source, name } = filter;

                    const isSelected = Object.prototype.hasOwnProperty.call(selectedSources, source);

                    const iconsData = getIconsData(homepage, source);

                    return (
                        <div key={name} className="d-flex align-items-center pb-1">
                            <Field
                                name={filter.id}
                                type="checkbox"
                                component={renderCheckboxField}
                                placeholder={t(name)}
                                disabled={isSelected}
                            />
                            {renderIcons(iconsData)}
                        </div>
                    );
                })}
            </div>
        );
    });

interface FormProps {
    t: (...args: unknown[]) => string;
    closeModal: (...args: unknown[]) => unknown;
    handleSubmit: (...args: unknown[]) => string;
    processingAddFilter: boolean;
    processingConfigFilter: boolean;
    whitelist?: boolean;
    modalType: string;
    toggleFilteringModal: (...args: unknown[]) => unknown;
    selectedSources?: object;
}

const Form = (props: FormProps) => {
    const {
        t,
        closeModal,
        handleSubmit,
        processingAddFilter,
        processingConfigFilter,
        whitelist,
        modalType,
        toggleFilteringModal,
        selectedSources,
    } = props;

    const openModal = (modalType: any, timeout = MODAL_OPEN_TIMEOUT) => {
        toggleFilteringModal();
        setTimeout(() => toggleFilteringModal({ type: modalType }), timeout);
    };

    const openFilteringListModal = () => openModal(MODAL_TYPE.CHOOSE_FILTERING_LIST);

    const openAddFiltersModal = () => openModal(MODAL_TYPE.ADD_FILTERS);

    return (
        <form onSubmit={handleSubmit}>
            <div className="modal-body modal-body--filters">
                {modalType === MODAL_TYPE.SELECT_MODAL_TYPE && (
                    <div className="d-flex justify-content-around">
                        <button
                            onClick={openFilteringListModal}
                            className="btn btn-success btn-standard mr-2 btn-large">
                            {t('choose_from_list')}
                        </button>

                        <button onClick={openAddFiltersModal} className="btn btn-primary btn-standard">
                            {t('add_custom_list')}
                        </button>
                    </div>
                )}
                {modalType === MODAL_TYPE.CHOOSE_FILTERING_LIST && renderFilters(filtersCatalog, selectedSources, t)}
                {modalType !== MODAL_TYPE.CHOOSE_FILTERING_LIST && modalType !== MODAL_TYPE.SELECT_MODAL_TYPE && (
                    <>
                        <div className="form__group">
                            <Field
                                id="name"
                                name="name"
                                type="text"
                                component={renderInputField}
                                className="form-control"
                                placeholder={t('enter_name_hint')}
                                normalizeOnBlur={(data: any) => data.trim()}
                            />
                        </div>

                        <div className="form__group">
                            <Field
                                id="url"
                                name="url"
                                type="text"
                                component={renderInputField}
                                className="form-control"
                                placeholder={t('enter_url_or_path_hint')}
                                validate={[validateRequiredValue, validatePath]}
                                normalizeOnBlur={(data: any) => data.trim()}
                            />
                        </div>

                        <div className="form__description">
                            {whitelist ? t('enter_valid_allowlist') : t('enter_valid_blocklist')}
                        </div>
                    </>
                )}
            </div>

            <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={closeModal}>
                    {t('cancel_btn')}
                </button>

                {modalType !== MODAL_TYPE.SELECT_MODAL_TYPE && (
                    <button
                        type="submit"
                        className="btn btn-success"
                        disabled={processingAddFilter || processingConfigFilter}>
                        {t('save_btn')}
                    </button>
                )}
            </div>
        </form>
    );
};

export default flow([withTranslation(), reduxForm({ form: FORM_NAME.FILTER })])(Form);
