"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useUIStore } from "@/store";
import {
  LayoutDashboard,
  Building2,
  Users,
  BarChart3,
  Activity,
  Settings,
  FileText,
  Shield,
  Bell,
  Flag,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

const navItems = [
  {
    title: "Dashboard",
    href: "/dashboard",
    icon: LayoutDashboard,
  },
  {
    title: "Tenants",
    href: "/tenants",
    icon: Building2,
  },
  {
    title: "Users",
    href: "/users",
    icon: Users,
  },
  {
    title: "Analytics",
    href: "/analytics",
    icon: BarChart3,
  },
  {
    title: "Notifications",
    href: "/notifications",
    icon: Bell,
    children: [
      { title: "Inbox", href: "/notifications" },
      { title: "Templates", href: "/notification-templates" },
    ],
  },
  {
    title: "System",
    href: "/system",
    icon: Activity,
    children: [
      { title: "Health", href: "/system/health" },
      { title: "Logs", href: "/system/logs" },
      { title: "Metrics", href: "/system/metrics" },
    ],
  },
  {
    title: "Audit Log",
    href: "/audit",
    icon: FileText,
  },
  {
    title: "Multi-Party",
    href: "/quorum",
    icon: Shield,
  },
  {
    title: "Feature Flags",
    href: "/feature-flags",
    icon: Flag,
  },
  {
    title: "Settings",
    href: "/settings",
    icon: Settings,
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const { sidebarOpen, toggleSidebar } = useUIStore();

  return (
    <aside
      className={cn(
        "fixed left-0 top-0 z-40 h-screen bg-white border-r transition-all duration-300",
        sidebarOpen ? "w-64" : "w-20"
      )}
    >
      <div className="flex flex-col h-full">
        {/* Logo */}
        <div className="flex items-center justify-between h-16 px-4 border-b">
          <Link href="/dashboard" className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <span className="text-white font-bold text-sm">R</span>
            </div>
            {sidebarOpen && (
              <span className="font-bold text-lg">RINCO</span>
            )}
          </Link>
          <button
            onClick={toggleSidebar}
            className="p-1.5 rounded-lg hover:bg-gray-100 transition-colors"
          >
            {sidebarOpen ? (
              <ChevronLeft className="w-5 h-5 text-gray-500" />
            ) : (
              <ChevronRight className="w-5 h-5 text-gray-500" />
            )}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
          {navItems.map((item) => (
            <NavItem
              key={item.href}
              item={item}
              isActive={pathname.startsWith(item.href)}
              isOpen={sidebarOpen}
            />
          ))}
        </nav>

        {/* Footer */}
        <div className="p-4 border-t">
          <div
            className={cn(
              "flex items-center gap-3 text-sm text-gray-500",
              !sidebarOpen && "justify-center"
            )}
          >
            <Shield className="w-4 h-4" />
            {sidebarOpen && <span>Admin Mode</span>}
          </div>
        </div>
      </div>
    </aside>
  );
}

function NavItem({
  item,
  isActive,
  isOpen,
}: {
  item: (typeof navItems)[0];
  isActive: boolean;
  isOpen: boolean;
}) {
  const Icon = item.icon;

  return (
    <div>
      <Link
        href={item.href}
        className={cn(
          "flex items-center gap-3 px-3 py-2 rounded-lg transition-colors",
          isActive
            ? "bg-primary/10 text-primary"
            : "text-gray-600 hover:bg-gray-100"
        )}
      >
        <Icon className="w-5 h-5 flex-shrink-0" />
        {isOpen && (
          <>
            <span className="font-medium">{item.title}</span>
          </>
        )}
      </Link>
    </div>
  );
}

export default Sidebar;
