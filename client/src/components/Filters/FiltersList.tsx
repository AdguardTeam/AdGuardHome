import React from 'react';
import classNames from 'classnames';
import { useFormContext } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

const getIconsData = (homepage: string, source: string) => [
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

const renderIcons = (iconsData: { iconName: string; href: string; className?: string }[]) =>
    iconsData.map(({ iconName, href, className = '' }) => (
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

type Filter = {
    categoryId: string;
    homepage: string;
    source: string;
    name: string;
};

type Category = {
    name: string;
    description: string;
};

type Props = {
    categories: Record<string, Category>;
    filters: Record<string, Filter>;
    selectedSources: Record<string, boolean>;
};

export const FiltersList = ({ categories, filters, selectedSources }: Props) => {
    const { t } = useTranslation();
    const { register } = useFormContext();

    return (
        <>
            {Object.entries(categories).map(([categoryId, category]) => {
                const categoryFilters = Object.entries(filters)
                    .filter(([, filter]) => filter.categoryId === categoryId)
                    .map(([key, filter]) => ({ ...filter, id: key }));

                return (
                    <div key={category.name} className="modal-body__item">
                        <h6 className="font-weight-bold mb-1">{t(category.name)}</h6>
                        <p className="mb-3">{t(category.description)}</p>
                        {categoryFilters.map((filter) => {
                            const { homepage, source, name, id } = filter;
                            const isSelected = selectedSources[source];
                            const iconsData = getIconsData(homepage, source);

                            return (
                                <div key={name} className="d-flex align-items-center pb-1">
                                    <label className="checkbox checkbox--settings">
                                        <span className="checkbox__marker" />
                                        <input
                                            id={id}
                                            type="checkbox"
                                            className="checkbox__input"
                                            disabled={isSelected}
                                            {...register(id)}
                                        />
                                        <span className="checkbox__label">
                                            <span className="checkbox__label-text">
                                                <span className="checkbox__label-title">{t(name)}</span>
                                            </span>
                                        </span>
                                    </label>
                                    {renderIcons(iconsData)}
                                </div>
                            );
                        })}
                    </div>
                );
            })}
        </>
    );
};
