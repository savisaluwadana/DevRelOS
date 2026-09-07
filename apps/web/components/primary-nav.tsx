"use client";

import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

type NavItem = {
  label: string;
  href: string;
};

function splitHref(href: string) {
  const [path, hash = ""] = href.split("#", 2);
  return { path: path || "/", hash: hash ? `#${hash}` : "" };
}

export function PrimaryNav({ items }: { items: NavItem[] }) {
  const pathname = usePathname();
  const [hash, setHash] = useState("");

  useEffect(() => {
    const update = () => setHash(window.location.hash);
    update();
    window.addEventListener("hashchange", update);
    return () => window.removeEventListener("hashchange", update);
  }, [pathname]);

  return (
    <nav className="nav-list" aria-label="Primary">
      {items.map((item) => {
        const target = splitHref(item.href);
        const active = pathname === target.path && (target.hash === "" || target.hash === hash);
        return (
          <a className={active ? "nav-item active" : "nav-item"} href={item.href} key={item.label} aria-current={active ? "page" : undefined}>
            <span className="nav-dot" />{item.label}
          </a>
        );
      })}
    </nav>
  );
}
