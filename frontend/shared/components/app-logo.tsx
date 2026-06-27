"use client";

import Image from "next/image";
import { useTheme } from "@/shared/components/theme-provider";
import { APP_BRAND_LOGO, APP_BRAND_LOGO_DARK, APP_BRAND_NAME } from "@/shared/brand";

type AppLogoProps = {
  alt?: string;
  width: number;
  height: number;
  priority?: boolean;
  className?: string;
};

export function AppLogo({
  alt = APP_BRAND_NAME,
  width,
  height,
  priority,
  className,
}: AppLogoProps) {
  const { resolvedTheme } = useTheme();

  return (
    <Image
      src={resolvedTheme === "dark" ? APP_BRAND_LOGO_DARK : APP_BRAND_LOGO}
      alt={alt}
      width={width}
      height={height}
      priority={priority}
      className={className}
    />
  );
}
