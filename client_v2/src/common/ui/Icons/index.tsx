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
    label: 'label',
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
            <path
                d="M13 14C13 14 15.5308 13.4189 17.0263 11.6741C18.5218 9.92934 19 7 19 7"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path
                d="M3 20L7.00509 11L11 20"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
            />
            <path d="M4 17.5H9.5" stroke="currentColor" strokeWidth="1.5" />
            <path d="M11.5 6.5H21.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M16.5 6V4.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
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
            <path d="M12 8V14" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
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
    </svg>
));
Icons.displayName = 'Icons';
