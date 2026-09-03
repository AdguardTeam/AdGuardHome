import { type JSX } from 'solid-js';
import { Link } from 'panel/common/ui/Link';
import { type QueryParams, type RoutePathKey } from 'panel/components/Routes/Paths';

import s from './StatCard.module.pcss';

type Props = {
    to: RoutePathKey;
    query?: QueryParams;
    children: JSX.Element;
};

export const StatLink = (props: Props) => (
    <Link to={props.to} query={props.query} class={s.statLink}>
        {props.children}
    </Link>
);
