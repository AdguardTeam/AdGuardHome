import {
  type JSX,
  createSignal,
  createEffect,
  onCleanup,
  Show,
} from "solid-js";
import cn from "clsx";
import { Icon } from "panel/common/ui/Icon";
import { Popover } from "@ark-ui/solid";

import "./Dropdown.pcss";
import s from "./Dropdown.module.pcss";

const TIMEOUT_HIDE_TOOLTIP = 1000;

type Props = {
  overlayClass?: string;
  menu: JSX.Element;
  position?: "bottomRight" | "bottomLeft" | "topRight" | "topLeft" | "top";
  trigger: "click" | "hover";
  noIcon?: true;
  iconClass?: string;
  class?: string;
  openClass?: string;
  open?: boolean;
  onOpenChange?: (e: boolean) => void;
  widthAuto?: boolean;
  flex?: boolean;
  minOverlayWidthMatchTrigger?: boolean;
  flexWrapper?: boolean;
  childrenClass?: string;
  wrapClass?: string;
  children?: JSX.Element;
  isSelect?: boolean;
  disableAnimation?: boolean;
  disabled?: boolean;
  autoClose?: boolean;
};

export const Dropdown = (props: Props) => {
  let timer: ReturnType<typeof setTimeout> | null = null;
  const [visible, setVisible] = createSignal(!!props.open);

  const onVisibleChange = (details: { open: boolean }) => {
    if (props.disabled) {
      return;
    }

    props.onOpenChange?.(details.open);
    setVisible(details.open);
  };

  createEffect(() => {
    if (typeof props.open === "boolean") {
      setVisible(props.open);
    }
  });

  onCleanup(() => {
    setVisible(false);
    if (timer) {
      clearTimeout(timer);
    }
  });

  const handleOverlayClick = () => {
    if (!props.autoClose) {
      return;
    }

    if (timer) {
      clearTimeout(timer);
    }
    timer = setTimeout(
      () => onVisibleChange({ open: false }),
      TIMEOUT_HIDE_TOOLTIP,
    );
  };

  // Ark UI uses floating-ui placement tokens. corvu collapsed every position
  // to plain `bottom`/`top` (ignoring left/right), so mapping `bottomRight` ->
  // `bottom-end` is an intentional behavior change that honors the `position`
  // prop. `flip` keeps the popover on-screen near viewport edges.
  // eslint-disable-next-line solid/reactivity
  const placement = () => {
    switch (props.position) {
      case "topRight":
        return "top-end";
      case "topLeft":
        return "top-start";
      case "bottomRight":
        return "bottom-end";
      case "bottomLeft":
        return "bottom-start";
      case "top":
        return "top";
      default:
        return "bottom-end";
    }
  };

  return (
    <Popover.Root
      open={visible()}
      onOpenChange={onVisibleChange}
      positioning={{
        placement: placement(),
        gutter: 4,
        flip: true,
      }}
      closeOnInteractOutside={true}
    >
      <Popover.Anchor>
        <div
          class={cn(
            props.class,
            s.wrapper,
            {
              [s.open]: props.flex,
              [s.disabled]: props.disabled,
            },
            visible() && props.openClass ? props.openClass : null,
            props.wrapClass,
          )}
        >
          {props.trigger === "click" ? (
            // Click mode MUST use Popover.Trigger: it tags the element with
            // `data-part="trigger"` so Ark UI excludes it from outside-click
            // detection. A plain div would race (outside-click closes, then the
            // click reopens). Ark UI has no `as` prop — use `asChild`.
            <Popover.Trigger
              asChild={(triggerProps) => (
                <div
                  {...triggerProps}
                  class={cn(props.childrenClass, {
                    [s.wrapper]: props.flexWrapper,
                  })}
                >
                  {props.children}
                </div>
              )}
            />
          ) : (
            // Hover mode: a plain div (no Trigger) so a click does NOT toggle.
            // Enter/leave go through onVisibleChange so `disabled` still guards.
            <div
              class={cn(props.childrenClass, {
                [s.wrapper]: props.flexWrapper,
              })}
              onMouseEnter={() => onVisibleChange({ open: true })}
              onMouseLeave={() => onVisibleChange({ open: false })}
            >
              {props.children}
            </div>
          )}
          <Show when={!props.noIcon}>
            <Icon
              class={cn(s.arrow, props.iconClass, { [s.active]: visible() })}
              icon="arrow_bottom"
            />
          </Show>
        </div>
      </Popover.Anchor>
      <Popover.Positioner>
        <Popover.Content
          class={cn(s.overlay, props.overlayClass, {
            [s.widthAuto]: props.widthAuto,
            [s.selectOverlay]: props.isSelect,
          })}
          onClick={handleOverlayClick}
        >
          {props.menu}
        </Popover.Content>
      </Popover.Positioner>
    </Popover.Root>
  );
};
