import React, { memo } from 'react';

import './Icons.pcss';

export const ICONS = {
    checkbox_off: 'checkbox_off',
    checkbox_on: 'checkbox_on',
    checkbox_plus: 'checkbox_plus',
    checkbox_minus: 'checkbox_minus',
    radio_on: 'radio_on',
    radio_off: 'radio_off',
    dashboard: 'dashboard',
    settings: 'settings',
    tune: 'tune',
    log: 'log',
    faq: 'faq',
    logout: 'logout',
    lang: 'lang',
    theme_auto: 'theme_auto',
    theme_dark: 'theme_dark',
    theme_light: 'theme_light',
    cross: 'cross',
    arrow_bottom: 'arrow_bottom',
    butter: 'butter',
    loader: 'loader',
    check: 'check',
    dot: 'dot',
    attention: 'attention',
    arrow: 'arrow',
    edit: 'edit',
    delete: 'delete',
    plus: 'plus',
    refresh: 'refresh',
    bullets: 'bullets',
    link: 'link',
    not_found_search: 'not_found_search',
    label: 'label',
    copy: 'copy',
    router: 'router',
    windows: 'windows',
    mac: 'mac',
    android: 'android',
    ios: 'ios',
    dns_privacy: 'dns_privacy',
    location: 'location',
    connections: 'connections',
    adblocking: 'adblocking',
    tracking: 'tracking',
    parental: 'parental',
    search: 'search',
    time: 'time',
    eye_open: 'eye_open',
    eye_close: 'eye_close',
} as const;

export type IconType = keyof typeof ICONS;

export const ICON_VALUES: IconType[] = Object.values(ICONS);

export const Icons = memo(() => (
    <svg xmlns="http://www.w3.org/2000/svg" className="icons">
        <symbol id="checkbox_off" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path
                d="M21 3H3V21H21V3Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="checkbox_on" viewBox="0 0 24 24" fillRule="evenodd" clipRule="evenodd">
            <path
                d="m22 22v-20h-20v20zm-4.4309-12.5115c.2698-.3143.2337-.7878-.0806-1.05759s-.7878-.23371-1.0576.08059l-5.4763 6.3798-3.41909-3.4869c-.29-.2957-.76485-.3004-1.06061-.0104-.29575.29-.30042.7649-.01041 1.0606l4.56351 4.6541z"
                fill="currentColor"
            />
        </symbol>

        <symbol id="checkbox_minus" viewBox="0 0 24 24" fillRule="evenodd" clipRule="evenodd">
            <path
                d="M19 2C20.6569 2 22 3.34315 22 5V19C22 20.6569 20.6569 22 19 22H5C3.34315 22 2 20.6569 2 19V5C2 3.34315 3.34315 2 5 2H19Z"
                fill="currentColor"
            />
            <path d="M6.40047 11.9854H17.6034" stroke="white" strokeWidth="1.5" strokeLinecap="round" />
        </symbol>

        <symbol id="checkbox_plus" viewBox="0 0 24 24" fillRule="evenodd" clipRule="evenodd">
            <path
                d="M21.25 5V19C21.25 20.2426 20.2426 21.25 19 21.25H5C3.75736 21.25 2.75 20.2426 2.75 19V5C2.75 3.75736 3.75736 2.75 5 2.75H19C20.2426 2.75 21.25 3.75736 21.25 5Z"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
            />
            <path d="M11.9854 6.40032V17.6033" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M6.40047 11.9854H17.6034" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </symbol>

        <symbol id="radio_off" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <circle
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="radio_on" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <circle
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <circle
                cx="12"
                cy="12"
                r="5"
                fill="currentColor"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="dashboard" viewBox="0 0 24 24" fill="none">
            <path
                d="M13.3589 3.48913C12.5707 2.83696 11.4293 2.83696 10.6411 3.48913L4.7709 8.34627C4.2826 8.7503 4 9.3506 4 9.98384V18.8733C4 20.0478 4.95354 21 6.12978 21H8.5C9.05228 21 9.5 20.5523 9.5 20V16.5C9.5 15.3954 10.3954 14.5 11.5 14.5H12H12.5C13.6046 14.5 14.5 15.3954 14.5 16.5V20C14.5 20.5523 14.9477 21 15.5 21H17.8702C19.0465 21 20 20.0478 20 18.8733V9.98384C20 9.3506 19.7174 8.7503 19.2291 8.34627L13.3589 3.48913Z"
                stroke="currentColor"
                strokeWidth="1.5"
            />
        </symbol>

        <symbol id="settings" viewBox="0 0 24 24" fill="none">
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M7.92921 18.9146C8.06451 18.9146 8.19778 18.9475 8.31752 19.0105C8.74311 19.2345 9.1883 19.4192 9.64753 19.5621C9.91308 19.6446 10.12 19.8541 10.1991 20.1207C10.453 20.9752 10.6939 21.6143 10.852 22H13.148C13.3061 21.6135 13.5475 20.9742 13.8011 20.1199C13.8803 19.8534 14.0872 19.6438 14.3527 19.5613C14.812 19.4184 15.2572 19.2337 15.6827 19.0097C15.9289 18.88 16.2236 18.8818 16.4682 19.0144C17.2521 19.4394 17.8745 19.7207 18.2592 19.8822L19.8832 18.2591C19.7215 17.874 19.4405 17.2513 19.0152 16.4673C18.8826 16.2227 18.8809 15.9281 19.0105 15.6819C19.2346 15.2563 19.4192 14.8111 19.5621 14.3518C19.6446 14.0863 19.8542 13.8794 20.1207 13.8002C20.9752 13.5463 21.6143 13.3054 22 13.1473V10.8514C21.6135 10.6933 20.9744 10.4519 20.12 10.1982C19.8534 10.119 19.6439 9.91214 19.5613 9.64659C19.4184 9.18735 19.2338 8.74215 19.0097 8.31656C18.8801 8.07037 18.8818 7.77571 19.0144 7.53109C19.4394 6.74718 19.7207 6.12475 19.8819 5.74009L18.2597 4.1168C17.8745 4.27827 17.2518 4.55954 16.4679 4.98483C16.2233 5.11742 15.9287 5.11918 15.6825 4.98952C15.2569 4.76548 14.8117 4.58084 14.3525 4.43792C14.0869 4.3554 13.88 4.14586 13.8009 3.87929C13.5472 3.02507 13.3061 2.3857 13.148 2H10.852C10.6958 2.38492 10.4546 3.02064 10.2009 3.86965C10.1242 4.14096 9.91549 4.35507 9.64623 4.4387C9.18701 4.58154 8.74182 4.76609 8.31621 4.99004C8.07003 5.1197 7.77538 5.11794 7.53076 4.98535C6.74686 4.56032 6.12443 4.27905 5.73978 4.11785L4.11678 5.74035C4.27824 6.12553 4.55951 6.74823 4.98479 7.53213C5.11738 7.77675 5.11913 8.07141 4.98948 8.3176C4.76553 8.7432 4.58099 9.1884 4.43815 9.64763C4.35563 9.91318 4.1461 10.1201 3.87952 10.1992C3.02505 10.4529 2.38596 10.6941 2 10.8521V13.1481C2.38387 13.3044 3.01568 13.5442 3.86208 13.7958C4.1443 13.8731 4.36551 14.0924 4.44518 14.374C4.58538 14.8197 4.76482 15.2522 4.98141 15.6663C5.12378 15.9228 5.12082 16.2352 4.97359 16.489C4.55508 17.2627 4.27746 17.8779 4.11756 18.2599L5.74082 19.8842C6.126 19.7228 6.74868 19.4415 7.53258 19.0162C7.65429 18.9499 7.79061 18.915 7.92921 18.9146Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinejoin="round"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 9.27273C10.4937 9.27273 9.27271 10.4938 9.27271 12C9.27271 13.5062 10.4937 14.7273 12 14.7273C13.5062 14.7273 14.7273 13.5062 14.7273 12C14.7255 10.4945 13.5055 9.27449 12 9.27273Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="tune" viewBox="0 0 24 24" fill="none">
            <circle
                cx="5"
                cy="19"
                r="2"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <circle
                cx="12"
                cy="12"
                r="2"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M19 3C20.1046 3 21 3.89543 21 5C21 6.10457 20.1046 7 19 7C17.8954 7 17 6.10457 17 5C17 3.89543 17.8954 3 19 3Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path d="M12 3V10" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M19 21V7" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M5 3V17" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M12 14V21" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </symbol>

        <symbol id="log" viewBox="0 0 24 24" fill="none">
            <path d="M4 6H20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M4 10H16" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M4 14H20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M4 18H12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </symbol>

        <symbol id="faq" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path
                d="M12 21C16.9706 21 21 16.9706 21 12C21 7.02944 16.9706 3 12 3C7.02944 3 3 7.02944 3 12C3 16.9706 7.02944 21 12 21Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M10 9.30278C10.0192 8.48184 11.002 7.7767 12.004 7.7767C13.006 7.7767 13.6034 8.16666 14.008 9.00001C14.3179 9.70342 14.0142 10.5459 12.9391 11.2797C12.0698 11.8368 11.7963 12.4126 11.7963 13.4587"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M11.7963 15.8952C11.8083 15.8972 11.7963 15.7042 11.7963 15.7042"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="logout" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path
                d="M15.5556 7L20 12"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M15.5556 17L20 12H8.80095"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M5 4L5 20H11"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M5 20L5 4H11"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="lang" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M13 14s2.53-.581 4.026-2.326S19 7 19 7" />
            <path stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M20 14s-2.53-.581-4.026-2.326S14 7 14 7M3 20l4.005-9L11 20" />
            <path stroke="currentColor" strokeWidth="1.5" d="M4 17.5h5.5" />
            <path stroke="currentColor" strokeLinecap="round" strokeWidth="1.5" d="M11.5 6.5h10M16.5 6V4.5" />
        </symbol>

        <symbol id="theme_auto" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 3C16.9706 3 21 7.02944 21 12C21 16.9706 16.9706 21 12 21C7.02944 21 3 16.9706 3 12C3 7.02944 7.02944 3 12 3Z"
                stroke="currentColor"
                strokeWidth="1.5"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 3V21C16.9706 21 21 16.9706 21 12C21 7.02944 16.9706 3 12 3Z"
                fill="currentColor"
                stroke="currentColor"
                strokeWidth="1.5"
            />
        </symbol>

        <symbol id="theme_dark" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path
                d="M3.80737 15.731L3.9895 15.0034C3.71002 14.9335 3.41517 15.0298 3.23088 15.2512C3.0466 15.4727 3.00545 15.7801 3.12501 16.0422L3.80737 15.731ZM14.1926 3.26892L14.3747 2.54137C14.0953 2.47141 13.8004 2.56772 13.6161 2.78917C13.4318 3.01062 13.3907 3.31806 13.5102 3.58018L14.1926 3.26892ZM12 20.2499C8.66479 20.2499 5.79026 18.2708 4.48974 15.4197L3.12501 16.0422C4.66034 19.4081 8.05588 21.7499 12 21.7499V20.2499ZM20.25 11.9999C20.25 16.5563 16.5563 20.2499 12 20.2499V21.7499C17.3848 21.7499 21.75 17.3847 21.75 11.9999H20.25ZM14.0105 3.99647C17.5955 4.89391 20.25 8.13787 20.25 11.9999H21.75C21.75 7.43347 18.6114 3.60193 14.3747 2.54137L14.0105 3.99647ZM13.5102 3.58018C13.9851 4.6211 14.25 5.77857 14.25 6.99995H15.75C15.75 5.5595 15.4371 4.1901 14.875 2.95766L13.5102 3.58018ZM14.25 6.99995C14.25 11.5563 10.5563 15.2499 5.99999 15.2499V16.7499C11.3848 16.7499 15.75 12.3847 15.75 6.99995H14.25ZM5.99999 15.2499C5.30559 15.2499 4.63225 15.1643 3.9895 15.0034L3.62525 16.4585C4.38616 16.649 5.18181 16.7499 5.99999 16.7499V15.2499Z"
                fill="currentColor"
            />
        </symbol>

        <symbol id="theme_light" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path
                d="M12 6.07692C8.73438 6.07692 6.07692 8.73438 6.07692 12C6.07692 15.2656 8.73438 17.9231 12 17.9231C15.2656 17.9231 17.9231 15.2656 17.9231 12C17.9231 8.73438 15.2656 6.07692 12 6.07692ZM12 16.2308C9.66313 16.2308 7.76923 14.3369 7.76923 12C7.76923 9.66313 9.66313 7.76923 12 7.76923C14.3369 7.76923 16.2308 9.66313 16.2308 12C16.2308 14.3369 14.3369 16.2308 12 16.2308ZM12 4.38462C12.4671 4.38462 12.8462 4.00559 12.8462 3.53846V1.84615C12.8462 1.37902 12.4671 1 12 1C11.5329 1 11.1538 1.37902 11.1538 1.84615V3.53846C11.1538 4.00559 11.5329 4.38462 12 4.38462ZM12 19.6154C11.5329 19.6154 11.1538 19.9944 11.1538 20.4615V22.1538C11.1538 22.621 11.5329 23 12 23C12.4671 23 12.8462 22.621 12.8462 22.1538V20.4615C12.8462 19.9944 12.4671 19.6154 12 19.6154ZM18.5809 6.6146L19.7774 5.41809C20.1079 5.08756 20.1079 4.5521 19.7774 4.22157C19.4468 3.89104 18.9114 3.89104 18.5809 4.22157L17.3843 5.41809C17.0538 5.74862 17.0538 6.28407 17.3843 6.6146C17.7149 6.94513 18.2503 6.94513 18.5809 6.6146ZM5.41914 17.3855L4.22263 18.582C3.8921 18.9124 3.8921 19.4479 4.22263 19.7784C4.55316 20.109 5.08862 20.109 5.41914 19.7784L6.61566 18.582C6.94619 18.2503 6.94619 17.7159 6.61566 17.3855C6.28518 17.0549 5.74967 17.0538 5.41914 17.3855ZM4.38462 12C4.38462 11.5329 4.00559 11.1538 3.53846 11.1538H1.84615C1.37902 11.1538 1 11.5329 1 12C1 12.4671 1.37902 12.8462 1.84615 12.8462H3.53846C4.00559 12.8462 4.38462 12.4671 4.38462 12ZM22.1538 11.1538H20.4615C19.9944 11.1538 19.6154 11.5329 19.6154 12C19.6154 12.4671 19.9944 12.8462 20.4615 12.8462H22.1538C22.621 12.8462 23 12.4671 23 12C23 11.5329 22.621 11.1538 22.1538 11.1538ZM5.41803 6.6146C5.74862 6.94513 6.28407 6.94513 6.61455 6.6146C6.94513 6.28407 6.94513 5.74862 6.61455 5.41809L5.41803 4.22157C5.0875 3.89104 4.5521 3.89104 4.22152 4.22157C3.89099 4.5521 3.89099 5.08756 4.22152 5.41809L5.41803 6.6146ZM18.582 17.3843C18.2503 17.0538 17.7159 17.0538 17.3855 17.3843C17.0549 17.7148 17.0538 18.2503 17.3855 18.5808L18.582 19.7773C18.9124 20.1078 19.4479 20.1078 19.7784 19.7773C20.109 19.4468 20.109 18.9113 19.7784 18.5808L18.582 17.3843Z"
                fill="currentColor"
            />
        </symbol>

        <symbol id="cross" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path d="M6.42857 6.42857L17.6043 17.6043" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M6.42871 17.5714L17.6045 6.39563" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </symbol>

        <symbol id="arrow_bottom" viewBox="0 0 24 24" fill="none" fillRule="evenodd" clipRule="evenodd">
            <path
                d="M6 10L12 16L18 10"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="butter" viewBox="0 0 24 24" fill="none">
            <path d="M4 12H20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M4 7H20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M4 17H20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </symbol>

        <symbol id="loader" viewBox="0 0 68 68" fill="none">
            <g stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="3">
                <path d="m17.8231 14.2872c.3222-.2648.6511-.5218.9863-.7708" />
                <path d="m24.2393 10.4348c.7506-.3113 1.5192-.58768 2.3039-.82726.4025-.1229.8093-.2361 1.22-.33934" />
                <path d="m34 8.5c14.0833 0 25.5 11.4167 25.5 25.5s-11.4167 25.5-25.5 25.5c-7.0416 0-13.4166-2.8542-18.0312-7.4688-2.3489-2.3488-4.2416-5.1538-5.534-8.2705" />
            </g>
        </symbol>

        <symbol id="check" viewBox="0 0 24 24" fill="currentColor" fillRule="evenodd" clipRule="evenodd">
            <path d="M17.7271 7.18879C18.0551 7.47031 18.0927 7.9644 17.8112 8.29236L10.9858 16.2439L6.22381 11.3874C5.9212 11.0788 5.92607 10.5833 6.23468 10.2807C6.5433 9.9781 7.03879 9.98297 7.3414 10.2916L10.9091 13.9301L16.6235 7.27288C16.9051 6.94492 17.3992 6.90727 17.7271 7.18879Z" />
        </symbol>

        <symbol id="dot" viewBox="0 0 24 24" fill="none">
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 13.5C11.1716 13.5 10.5 12.8284 10.5 12C10.5 11.1716 11.1716 10.5 12 10.5C12.8284 10.5 13.5 11.1716 13.5 12C13.5 12.8284 12.8284 13.5 12 13.5Z"
                fill="currentColor"
            />
        </symbol>

        <symbol id="attention" viewBox="0 0 24 24" fill="none">
            <circle
                cx="9"
                cy="9"
                r="9"
                transform="matrix(1 0 0 -1 3 21)"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M12 8V14"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M11.997 16.4045C12.009 16.4025 11.997 16.5955 11.997 16.5955"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="label" viewBox="0 0 24 24" fill="none">
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 8C9.79086 8 8 9.79086 8 12C8 14.2091 9.79086 16 12 16C14.2091 16 16 14.2091 16 12C15.9974 9.79193 14.2081 8.00258 12 8Z"
                fill="currentColor"
            />
        </symbol>

        <symbol id="arrow" viewBox="0 0 24 24" fill="none">
            <path
                d="M9 18L15 12L9 6"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="edit" viewBox="0 0 24 24" fill="none">
            <path
                d="M4 20.5H19.9828"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M10.7773 16.435L6.5347 12.1924L13.6058 5.12132C14.7773 3.94975 16.6768 3.94975 17.8484 5.12132V5.12132C19.02 6.29289 19.02 8.19239 17.8484 9.36396L10.7773 16.435Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M5.12087 17.8492L6.53508 12.1924L10.7777 16.435L5.12087 17.8492Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M16.7891 9.0104L13.9607 6.18197"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="delete" viewBox="0 0 24 24" fill="none">
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M7 9H17L16.2367 19.0755C16.1972 19.597 15.7625 20 15.2396 20H8.76044C8.23746 20 7.80281 19.597 7.7633 19.0755L7 9Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path d="M6 6.5H18" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path
                d="M14 6V4L10 4V6"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M13.5 12V17V12Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M10.5 12V17V12Z"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="plus" viewBox="0 0 24 24" fill="none">
            <path d="M4 12H20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M12 4L12 20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </symbol>

        <symbol id="refresh" viewBox="0 0 24 24" fill="none">
            <path
                d="M16.9111 16.1024C15.7416 17.4924 13.9892 18.3756 12.0305 18.3756C9.67057 18.3756 7.61013 17.0935 6.50777 15.1878H7.6973L4.84867 10.2538L2 15.1878H4.69083C5.92255 18.0198 8.7452 20 12.0305 20C14.3141 20 16.3742 19.0432 17.8319 17.5085L16.9111 16.1024Z"
                fill="currentColor"
            />
            <path
                d="M6.08645 9.68933C7.0119 7.31041 9.32428 5.62437 12.0305 5.62437C15.1493 5.62437 17.745 7.86377 18.2975 10.8223H16.0508L18.8994 15.7563L21.7481 10.8223H19.9444C19.3749 6.96241 16.0486 4 12.0305 4C8.96982 4 6.3106 5.71875 4.96534 8.24372L6.08645 9.68933Z"
                fill="currentColor"
            />
        </symbol>

        <symbol id="bullets" viewBox="0 0 24 24" fill="none">
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 7C11.1716 7 10.5 6.32843 10.5 5.5C10.5 4.67157 11.1716 4 12 4C12.8284 4 13.5 4.67157 13.5 5.5C13.5 6.32843 12.8284 7 12 7Z"
                fill="currentColor"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 13.5C11.1716 13.5 10.5 12.8284 10.5 12C10.5 11.1716 11.1716 10.5 12 10.5C12.8284 10.5 13.5 11.1716 13.5 12C13.5 12.8284 12.8284 13.5 12 13.5Z"
                fill="currentColor"
            />
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M12 20C11.1716 20 10.5 19.3284 10.5 18.5C10.5 17.6716 11.1716 17 12 17C12.8284 17 13.5 17.6716 13.5 18.5C13.5 19.3284 12.8284 20 12 20Z"
                fill="currentColor"
            />
        </symbol>

        <symbol id="link" viewBox="0 0 24 24" fill="none">
            <path
                d="M18 11.8333V6.00001L12.2857 6.00001"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M17.9219 6.03911L11.1328 12.9696"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M8.76672 7H8C6.89543 7 6 7.89543 6 9V16C6 17.1046 6.89543 18 8 18H15C16.1046 18 17 17.1046 17 16V15.2961"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
        </symbol>

        <symbol id="not_found_search" viewBox="0 0 64 64" fill="none">
            <path
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="1.5"
                d="M39.631 24.31a15.16 15.16 0 0 1-4.196 12.723c-5.872 5.999-15.505 6.099-21.504.226-5.998-5.872-6.107-15.505-.226-21.504 4.377-4.476 10.838-5.672 16.33-3.552M35.435 37.033 55 56.172"
            />
            <path
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="1.5"
                d="M39.522 21.564a7.023 7.023 0 1 0 0-14.046 7.023 7.023 0 0 0 0 14.046ZM35.753 10.645l7.54 7.54M35.753 18.157l7.54-7.54"
            />
        </symbol>
        <symbol id="copy" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
            <g fill="none" fillRule="evenodd" stroke="currentColor" strokeWidth="1.5"><rect height="12" rx="2" width="12" x="4" y="8" /><path d="m8 6.12528887v-.62528887c0-.82842712.67157288-1.5 1.5-1.5h8.5c1.1045695 0 2 .8954305 2 2v8c0 1.1045695-.8954305 2-2 2h-.265606" />
            </g>
        </symbol>
        <symbol id="router" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path fillRule="evenodd" clipRule="evenodd" d="M4.48705 13.316C3.11348 13.316 2 14.3645 2 15.658C2 16.9514 3.11348 18 4.48705 18H19.513C20.8865 18 22 16.9514 22 15.658C22 14.3645 20.8865 13.316 19.513 13.316H4.48705ZM19.1503 16.6338C19.7512 16.6338 20.2383 16.1751 20.2383 15.6092C20.2383 15.0433 19.7512 14.5846 19.1503 14.5846C18.5493 14.5846 18.0622 15.0433 18.0622 15.6092C18.0622 16.1751 18.5493 16.6338 19.1503 16.6338Z" fill="currentColor" />
            <path d="M7.03453 11.4077C7.31379 11.6695 7.77344 11.6344 8.04807 11.3683C9.05991 10.3878 10.4753 9.77871 12.0419 9.77871C13.5715 9.77871 14.9571 10.3595 15.9635 11.2996C16.24 11.5579 16.6928 11.5882 16.9688 11.3294C17.2043 11.1087 17.2263 10.7536 16.9958 10.5283C15.7445 9.30551 13.9875 8.54485 12.0419 8.54485C10.0504 8.54485 8.25666 9.34172 7.0005 10.6153C6.77768 10.8412 6.8025 11.1901 7.03453 11.4077Z" fill="currentColor" />
            <path d="M5.12672 9.61912C5.39921 9.87457 5.84491 9.85132 6.11303 9.5918C7.6157 8.13733 9.71674 7.23387 12.0419 7.23387C14.3286 7.23387 16.3986 8.10774 17.8958 9.52021C18.1651 9.77431 18.6061 9.79449 18.8763 9.54119C19.1183 9.31432 19.1348 8.94827 18.8962 8.71824C17.1564 7.04111 14.7283 6 12.0419 6C9.31095 6 6.84689 7.0759 5.10159 8.80213C4.86843 9.03274 4.88709 9.39447 5.12672 9.61912Z" fill="currentColor" />
        </symbol>
        <symbol id="windows" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path fillRule="evenodd" clipRule="evenodd" d="M11.1459 12.4343C11.1509 14.4332 11.1569 17.8293 11.1619 20.4688C14.7759 20.9576 18.39 21.4613 21.998 22C21.998 18.8487 22.002 15.7115 21.998 12.7131C18.381 12.7131 14.7649 12.4343 11.1459 12.4343ZM2 12.4353V19.2215C4.72581 19.5893 7.45163 19.9411 10.1724 20.3429C10.1774 17.7174 10.1704 15.0908 10.1704 12.4652C7.44662 12.4702 4.72381 12.4263 2 12.4353ZM2 4.84344V11.6107C4.72581 11.6177 7.45163 11.5767 10.1774 11.5797C10.1754 8.96017 10.1754 6.34361 10.1724 3.72405C7.44461 4.06486 4.71679 4.42567 2 4.84344ZM22 11.4718C18.388 11.4858 14.7759 11.5408 11.1619 11.5517C11.1599 8.88921 11.1599 6.22967 11.1619 3.56914C14.7689 3.01844 18.384 2.50072 21.998 2C22 5.15826 21.998 8.31353 22 11.4718Z" fill="currentColor" />
            <mask id="mask0_10282_40" maskUnits="userSpaceOnUse" x="2" y="2" width="20" height="20">
                <path fillRule="evenodd" clipRule="evenodd" d="M11.1459 12.4343C11.1509 14.4332 11.1569 17.8293 11.1619 20.4688C14.7759 20.9576 18.39 21.4613 21.998 22C21.998 18.8487 22.002 15.7115 21.998 12.7131C18.381 12.7131 14.7649 12.4343 11.1459 12.4343ZM2 12.4353V19.2215C4.72581 19.5893 7.45163 19.9411 10.1724 20.3429C10.1774 17.7174 10.1704 15.0908 10.1704 12.4652C7.44662 12.4702 4.72381 12.4263 2 12.4353ZM2 4.84344V11.6107C4.72581 11.6177 7.45163 11.5767 10.1774 11.5797C10.1754 8.96017 10.1754 6.34361 10.1724 3.72405C7.44461 4.06486 4.71679 4.42567 2 4.84344ZM22 11.4718C18.388 11.4858 14.7759 11.5408 11.1619 11.5517C11.1599 8.88921 11.1599 6.22967 11.1619 3.56914C14.7689 3.01844 18.384 2.50072 21.998 2C22 5.15826 21.998 8.31353 22 11.4718Z" fill="currentColor" />
            </mask>
            <g mask="url(#mask0_10282_40)">
            </g>
        </symbol>
        <symbol id="mac" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path fillRule="evenodd" clipRule="evenodd" d="M12.0001 2.11914C6.48628 2.11914 2.11914 6.48628 2.11914 12.0001C2.11914 17.5144 6.48628 21.881 12.0001 21.881C17.5139 21.881 21.881 17.5144 21.881 12.0001C21.881 6.48628 17.5144 2.11914 12.0001 2.11914ZM13.1429 7.59353V7.39806L12.5353 7.43978C12.3633 7.45064 12.2341 7.48722 12.1472 7.54895C12.0604 7.61125 12.0169 7.69755 12.0169 7.80786C12.0169 7.9153 12.0598 8.00104 12.1461 8.06448C12.2318 8.12849 12.3473 8.15992 12.4913 8.15992C12.5833 8.15992 12.6696 8.14563 12.7491 8.11763C12.8285 8.08962 12.8982 8.05019 12.9565 7.99989C13.0148 7.95017 13.0606 7.89073 13.0931 7.82157C13.1263 7.75242 13.1429 7.6764 13.1429 7.59353ZM14.2499 6.75395C14.3059 6.59678 14.3848 6.46189 14.4871 6.34987C14.5894 6.23785 14.7122 6.15155 14.8563 6.09097C15.0003 6.03038 15.1609 6.00009 15.3375 6.00009C15.497 6.00009 15.641 6.0241 15.769 6.07153C15.8976 6.11897 16.0074 6.18298 16.0994 6.26357C16.1914 6.34416 16.2645 6.43789 16.3183 6.54534C16.372 6.65279 16.404 6.76652 16.4149 6.88598H15.9388C15.9268 6.82025 15.905 6.75852 15.8742 6.70194C15.8433 6.64536 15.8027 6.59621 15.7524 6.55448C15.7016 6.51276 15.6421 6.48018 15.5741 6.45675C15.5055 6.43275 15.429 6.42132 15.3427 6.42132C15.2415 6.42132 15.1495 6.44189 15.0677 6.48247C14.9854 6.52305 14.9151 6.58077 14.8568 6.65507C14.7985 6.72937 14.7534 6.82025 14.7214 6.92656C14.6888 7.03343 14.6728 7.15174 14.6728 7.28205C14.6728 7.41694 14.6888 7.5381 14.7214 7.64441C14.7534 7.75129 14.7991 7.84102 14.8586 7.91475C14.9174 7.98847 14.9889 8.04506 15.0717 8.08392C15.1546 8.12279 15.2461 8.14279 15.3455 8.14279C15.5084 8.14279 15.641 8.1045 15.7439 8.02791C15.8468 7.95132 15.913 7.8393 15.9439 7.69185H16.4206C16.4069 7.82216 16.3709 7.94104 16.3126 8.04849C16.2543 8.15593 16.1788 8.24738 16.0862 8.3234C15.9931 8.39941 15.8839 8.45828 15.7582 8.49943C15.6324 8.54058 15.4941 8.56173 15.3438 8.56173C15.1655 8.56173 15.0049 8.53201 14.8603 8.47314C14.7162 8.41427 14.5922 8.32911 14.4893 8.2188C14.3865 8.1085 14.307 7.97419 14.251 7.81587C14.195 7.65755 14.167 7.4798 14.167 7.28148C14.1659 7.08773 14.1939 6.9117 14.2499 6.75395ZM7.47548 6.03264H7.95214V6.45615H7.96129C7.99043 6.38585 8.02873 6.32241 8.07559 6.26755C8.12246 6.21211 8.17618 6.16524 8.23791 6.1258C8.29907 6.08637 8.36708 6.05665 8.44024 6.03607C8.51396 6.0155 8.59169 6.00521 8.67285 6.00521C8.84774 6.00521 8.99577 6.04693 9.11579 6.13038C9.23639 6.21382 9.32212 6.33384 9.37241 6.49045H9.38442C9.41642 6.415 9.45872 6.34756 9.51073 6.28812C9.56274 6.22868 9.62275 6.17724 9.69019 6.13495C9.75763 6.09265 9.83193 6.06065 9.91252 6.03836C9.9931 6.01607 10.0777 6.00521 10.1669 6.00521C10.2897 6.00521 10.4012 6.02464 10.5018 6.06408C10.6024 6.10351 10.6881 6.15838 10.7595 6.22982C10.831 6.30127 10.8858 6.38814 10.9241 6.48987C10.9624 6.59161 10.9819 6.70477 10.9819 6.82937V8.53312H10.4846V6.94882C10.4846 6.78479 10.4423 6.65733 10.3577 6.5676C10.2737 6.47787 10.1531 6.43272 9.99653 6.43272C9.91995 6.43272 9.84965 6.44644 9.78564 6.4733C9.7222 6.50016 9.66676 6.53788 9.62103 6.58646C9.57474 6.63447 9.53873 6.69277 9.51301 6.76021C9.48672 6.82765 9.47358 6.90138 9.47358 6.9814V8.53312H8.9832V6.90767C8.9832 6.83565 8.97177 6.7705 8.94948 6.7122C8.92719 6.6539 8.89575 6.60418 8.85403 6.56246C8.81288 6.52074 8.76201 6.4893 8.70314 6.46701C8.6437 6.44472 8.5774 6.43329 8.50368 6.43329C8.42709 6.43329 8.35622 6.44758 8.29049 6.47616C8.22534 6.50473 8.16933 6.54417 8.12246 6.59447C8.07559 6.64533 8.03902 6.70477 8.0133 6.77393C7.98815 6.84251 7.947 6.91796 7.947 6.99911V8.53255H7.47548V6.03264ZM8.55738 18.0001C6.37124 18.0001 5.00012 16.4809 5.00012 14.0565C5.00012 11.632 6.37124 10.1077 8.55738 10.1077C10.7435 10.1077 12.1095 11.632 12.1095 14.0565C12.1095 16.4804 10.7435 18.0001 8.55738 18.0001ZM12.605 8.52686C12.5244 8.54744 12.4421 8.55773 12.3575 8.55773C12.2329 8.55773 12.1192 8.54001 12.0157 8.50457C11.9117 8.46914 11.8231 8.41941 11.7488 8.35483C11.6745 8.29025 11.6162 8.21252 11.5751 8.12107C11.5333 8.02962 11.5128 7.92789 11.5128 7.81587C11.5128 7.5964 11.5945 7.42494 11.7579 7.30148C11.9214 7.17803 12.158 7.10602 12.4684 7.08601L13.1428 7.04715V6.85397C13.1428 6.70994 13.0971 6.59964 13.0056 6.52476C12.9142 6.44989 12.785 6.41217 12.6175 6.41217C12.5501 6.41217 12.4867 6.42074 12.4284 6.43732C12.3701 6.45446 12.3186 6.47847 12.274 6.5099C12.2295 6.54134 12.1923 6.57906 12.1637 6.62307C12.1346 6.66651 12.1146 6.71566 12.1037 6.76938H11.6362C11.6391 6.65908 11.6665 6.55677 11.7179 6.46304C11.7694 6.36931 11.8391 6.28815 11.9277 6.21899C12.0163 6.14983 12.1197 6.09611 12.2398 6.05782C12.3598 6.01952 12.4901 6.00009 12.6313 6.00009C12.7833 6.00009 12.921 6.01895 13.0445 6.05782C13.1679 6.09668 13.2737 6.15098 13.3611 6.22242C13.4486 6.29386 13.516 6.37959 13.5634 6.48018C13.6109 6.58077 13.6349 6.69337 13.6349 6.81739V8.53258H13.1588V8.11593H13.1468C13.1114 8.18337 13.0668 8.24452 13.0125 8.29882C12.9576 8.35312 12.8965 8.39998 12.8284 8.43827C12.7599 8.47657 12.6856 8.50629 12.605 8.52686ZM15.7502 18.0001C14.083 18.0001 12.9496 17.1268 12.8708 15.7557H13.9561C14.0407 16.5392 14.798 17.0582 15.8353 17.0582C16.8304 17.0582 17.5454 16.5392 17.5454 15.83C17.5454 15.2161 17.111 14.8452 16.1057 14.5914L15.1261 14.348C13.7178 13.9988 13.0777 13.3581 13.0777 12.3047C13.0777 11.0079 14.2104 10.1077 15.825 10.1077C17.4025 10.1077 18.5033 11.013 18.5456 12.315H17.4711C17.3968 11.5314 16.7567 11.0496 15.8033 11.0496C14.8557 11.0496 14.1996 11.5366 14.1996 12.2407C14.1996 12.7962 14.6122 13.1249 15.6181 13.3786L16.444 13.585C18.0163 13.9662 18.6622 14.5857 18.6622 15.6974C18.6616 17.1159 17.5397 18.0001 15.7502 18.0001ZM8.55741 11.0811C7.05941 11.0811 6.12266 12.2299 6.12266 14.0559C6.12266 15.8769 7.05941 17.0256 8.55741 17.0256C10.0503 17.0256 10.9922 15.8769 10.9922 14.0559C10.9927 12.2299 10.0503 11.0811 8.55741 11.0811Z" fill="currentColor" />
        </symbol>
        <symbol id="android" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path fillRule="evenodd" clipRule="evenodd" d="M6.6 17.2502C6.6 17.7314 7.005 18.1252 7.5 18.1252H8.4V21.1877C8.4 21.9139 9.003 22.5002 9.75 22.5002C10.497 22.5002 11.1 21.9139 11.1 21.1877V18.1252H12.9V21.1877C12.9 21.9139 13.503 22.5002 14.25 22.5002C14.997 22.5002 15.6 21.9139 15.6 21.1877V18.1252H16.5C16.995 18.1252 17.4 17.7314 17.4 17.2502V8.5002H6.6V17.2502ZM3.85 8.5002C3.103 8.5002 2.5 9.08645 2.5 9.8127V15.9377C2.5 16.6639 3.103 17.2502 3.85 17.2502C4.597 17.2502 5.2 16.6639 5.2 15.9377V9.8127C5.2 9.08645 4.597 8.5002 3.85 8.5002ZM20.15 8.5002C19.403 8.5002 18.8 9.08645 18.8 9.8127V15.9377C18.8 16.6639 19.403 17.2502 20.15 17.2502C20.897 17.2502 21.5 16.6639 21.5 15.9377V9.8127C21.5 9.08645 20.897 8.5002 20.15 8.5002ZM15.177 3.0902L16.347 1.9527C16.527 1.7777 16.527 1.50645 16.347 1.33145C16.167 1.15645 15.888 1.15645 15.708 1.33145L14.376 2.62645C13.665 2.27645 12.855 2.0752 12 2.0752C11.136 2.0752 10.326 2.27645 9.60598 2.62645L8.26498 1.33145C8.08498 1.15645 7.80598 1.15645 7.62598 1.33145C7.44598 1.50645 7.44598 1.7777 7.62598 1.9527L8.80498 3.09895C7.47298 4.0527 6.59998 5.58395 6.59998 7.3252H17.4C17.4 5.58395 16.527 4.04395 15.177 3.0902ZM10.2 5.5752H9.29999V4.7002H10.2V5.5752ZM14.7 5.5752H13.8V4.7002H14.7V5.5752Z" fill="currentColor" />
            <mask id="mask0_10282_276" maskUnits="userSpaceOnUse" x="6" y="8" width="12" height="15">
                <path fillRule="evenodd" clipRule="evenodd" d="M6.6001 17.25C6.6001 17.7312 7.0051 18.125 7.5001 18.125H8.4001V21.1875C8.4001 21.9137 9.0031 22.5 9.7501 22.5C10.4971 22.5 11.1001 21.9137 11.1001 21.1875V18.125H12.9001V21.1875C12.9001 21.9137 13.5031 22.5 14.2501 22.5C14.9971 22.5 15.6001 21.9137 15.6001 21.1875V18.125H16.5001C16.9951 18.125 17.4001 17.7312 17.4001 17.25V8.5H6.6001V17.25Z" fill="currentColor" />
            </mask>
            <g mask="url(#mask0_10282_276)">
            </g>
        </symbol>
        <symbol id="ios" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path fillRule="evenodd" clipRule="evenodd" d="M15.2516 3.51225C16.0236 2.58276 16.5439 1.28808 16.4015 0C15.2891 0.0439993 13.9432 0.738089 13.1456 1.66647C12.4293 2.49036 11.804 3.80704 11.9721 5.06982C13.213 5.16552 14.4796 4.44283 15.2516 3.51225ZM18.0342 11.6873C18.0653 15.017 20.9679 16.1247 21 16.139C20.9764 16.2171 20.5364 17.7175 19.4711 19.2684C18.5492 20.6082 17.5931 21.9425 16.0867 21.9711C14.607 21.9986 14.1305 21.0977 12.4378 21.0977C10.7461 21.0977 10.2172 21.9425 8.81675 21.9986C7.36277 22.0525 6.25462 20.5488 5.32634 19.2134C3.42696 16.4822 1.97619 11.4949 3.92482 8.1289C4.89271 6.45803 6.62186 5.39875 8.49983 5.37235C9.92704 5.34485 11.275 6.32823 12.1476 6.32823C13.0202 6.32823 14.6584 5.14575 16.38 5.31955C17.1006 5.34925 19.1242 5.60884 20.4229 7.50191C20.318 7.56681 18.0085 8.90439 18.0342 11.6873Z" fill="currentColor" />
        </symbol>
        <symbol id="dns_privacy" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path fillRule="evenodd" clipRule="evenodd" d="M3 7.10913C4.76946 4.05983 8.064 2 11.8362 2C15.6085 2 18.9022 4.05983 20.6733 7.10913H18.639C17.7444 5.91931 16.5552 4.98311 15.1887 4.39277C15.6357 5.15063 16.007 6.07282 16.2948 7.10913H14.5006C13.8279 5.01524 12.7993 3.70304 11.8362 3.70304C10.8732 3.70304 9.84453 5.01524 9.17182 7.10913H7.37682C7.66548 6.07282 8.0376 5.15063 8.4838 4.39277C7.11787 4.9834 5.92931 5.91957 5.03514 7.10913H3ZM15.8986 11.3419C15.8986 10.395 16.6581 9.77343 17.8264 9.77343H17.8273C18.9615 9.77343 19.7483 10.3865 19.7687 11.2832H18.6311C18.6039 10.9409 18.2786 10.7024 17.8452 10.7024C17.4126 10.7024 17.1265 10.9085 17.1265 11.2304C17.1265 11.4952 17.3402 11.6502 17.8511 11.7507L18.4761 11.8716C19.4349 12.0572 19.8641 12.4906 19.8641 13.2596C19.8641 14.2703 19.0909 14.8809 17.8213 14.8809C16.5951 14.8809 15.8228 14.3027 15.8032 13.3686H16.9783C17.0081 13.7219 17.3606 13.9476 17.8716 13.9476C18.3314 13.9476 18.6464 13.7262 18.6464 13.4069C18.6464 13.1386 18.4353 12.9939 17.8818 12.8832L17.2431 12.7555C16.3567 12.5877 15.8986 12.1049 15.8986 11.3419ZM4.04497 9.89989V14.7502H6.0801C7.53791 14.7502 8.34685 13.8705 8.34685 12.2782C8.34685 10.7633 7.52428 9.89989 6.0801 9.89989H4.04497ZM9.9484 14.7501V9.90325V9.9024H10.9311L12.9509 12.6298H13.0156V9.9024H14.1847V14.7501H13.2063L11.1823 11.9954H11.1175V14.7501H9.9484ZM5.27703 10.8911H5.85777C6.63436 10.8911 7.09162 11.3986 7.09162 12.302C7.09162 13.2625 6.65735 13.7607 5.85692 13.7607H5.27703V10.8911ZM11.8362 22.4365C8.06401 22.4365 4.76948 20.3767 3.00002 17.3274H5.03515C5.92936 18.5173 7.11829 19.4535 8.48466 20.0438C8.03846 19.2859 7.66635 18.3637 7.37768 17.3274H9.17439C9.8471 19.4213 10.8757 20.7335 11.8388 20.7335C12.8019 20.7335 13.8288 19.4213 14.5015 17.3274H16.2948C16.0078 18.3637 15.6366 19.2859 15.1904 20.0438C16.556 19.453 17.7443 18.5168 18.6382 17.3274H20.6725C18.903 20.3767 15.6085 22.4365 11.8362 22.4365Z" fill="currentColor" />
        </symbol>
        <symbol
            id="location"
            fill="none"
            height="24"
            viewBox="0 0 24 24"
            width="24"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="1.5"
        >
            <path clipRule="evenodd" d="m4 9.36842c0-4.06947 3.58172-7.36842 8-7.36842 4.4183 0 8 3.29895 8 7.36842 0 5.41278-8 12.63158-8 12.63158s-8-7.2188-8-12.63158z" fillRule="evenodd" />
            <circle cx="12" cy="10" r="3" />
        </symbol>
        <symbol
            id="connections"
            fill="none"
            height="24"
            viewBox="0 0 24 24"
            width="24"
            xmlns="http://www.w3.org/2000/svg"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="1.5"
        >
            <path d="m15.5 4 3 3" />
            <path d="m15.5 10 3-3" />
            <path d="m18.5 7h-13" />
            <path d="m8.5 14-3 3" />
            <path d="m8.5 20-3-3" />
            <path d="m5.5 17h13" />
        </symbol>
        <symbol
            id="adblocking"
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
        >
            <path fillRule="evenodd" clipRule="evenodd" d="M3 8C3 7.44772 3.44772 7 4 7H20C20.5523 7 21 7.44772 21 8V16C21 16.5523 20.5523 17 20 17H4C3.44772 17 3 16.5523 3 16V8Z" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M4 21L20 3" strokeWidth="1.5" strokeLinecap="round" />
        </symbol>
        <symbol
            id="tracking"
            fill="none"
            height="24"
            viewBox="0 0 24 24"
            width="24"
            xmlns="http://www.w3.org/2000/svg"
        >
            <path d="m17.3918 9.16765c.8694.69904 1.7388 1.56715 2.6082 2.60435-2.6667 3.3333-5.3333 5-8 5-.4393 0-.8786-.0453-1.318-.1357m-2.51032-1.0233c-1.0044-.6589-2.39496-1.9393-4.17168-3.841 2.66667-3.18135 5.33333-4.772 8-4.772 1.0715 0 2.143.25682 3.2145.77047m-11.2145 13.22953 16-18m-9.4142 10.4142c-.3619-.3619-.5858-.8619-.5858-1.4142 0-1.1046.8954-2 2-2 .4707 0 .9035.1626 1.2451.4348z" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" />
        </symbol>
        <symbol
            id="parental"
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
        >
            <circle cx="6.76196" cy="17.2381" r="2.66005" transform="rotate(-135 6.76196 17.2381)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <ellipse cx="15.0385" cy="8.96143" rx="3.2162" ry="5.21453" transform="rotate(-135 15.0385 8.96143)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <ellipse cx="9.9898" cy="14.0103" rx="7.07525" ry="1.91841" transform="rotate(-135 9.9898 14.0103)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </symbol>
        <symbol
            id="search"
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
        >
            <circle cx="9.5" cy="9.5" r="5.5" strokeWidth="1.5" />
            <path d="M14 14L19 19" strokeWidth="1.5" strokeLinecap="round" />
        </symbol>
        <symbol
            id="time"
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
        >
            <circle cx="12" cy="12" r="9" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M16 9L12 13.5L8.5 11.5" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </symbol>
        <symbol
            id="eye_opened"
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
        >
            <path fillRule="evenodd" clipRule="evenodd" d="M4 11.772C6.66667 8.59065 9.33333 7 12 7C14.6667 7 17.3333 8.59065 20 11.772C20 11.772 16 16.772 12 16.772C8 16.772 4 11.772 4 11.772Z" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path fillRule="evenodd" clipRule="evenodd" d="M12 10C13.1046 10 14 10.8954 14 12C14 13.1046 13.1046 14 12 14C10.8954 14 10 13.1046 10 12C10 10.8954 10.8954 10 12 10Z" strokeWidth="1.5" />
        </symbol>
        <symbol id="eye_close" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M4 10C4 10 8 14.772 12 14.772C16 14.772 20 10 20 10" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M12 15V17" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M18 13V15" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M6 13V15" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </symbol>
        <symbol id="eye_open" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path fillRule="evenodd" clipRule="evenodd" d="M4 11.772C6.66667 8.59065 9.33333 7 12 7C14.6667 7 17.3333 8.59065 20 11.772C20 11.772 16 16.772 12 16.772C8 16.772 4 11.772 4 11.772Z" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            <path fillRule="evenodd" clipRule="evenodd" d="M12 10C13.1046 10 14 10.8954 14 12C14 13.1046 13.1046 14 12 14C10.8954 14 10 13.1046 10 12C10 10.8954 10.8954 10 12 10Z" stroke="currentColor" strokeWidth="1.5" />
        </symbol>
    </svg>
));

Icons.displayName = 'Icons';
