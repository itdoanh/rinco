"use client";

import React, { Component, type ReactNode, type ErrorInfo } from "react";
import { AlertCircle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";

interface ErrorBoundaryProps {
  children: ReactNode;
  /** Tùy chọn fallback; nếu không sẽ dùng default UI. */
  fallback?: ReactNode;
  /** Callback khi có lỗi (cho logging / Sentry). */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  /** Label hiển thị trên header fallback. */
  title?: string;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
}

/**
 * ErrorBoundary — React class component bắt mọi render error trong
 * subtree. Khi xảy ra lỗi sẽ hiển thị fallback UI với nút "Try again"
 * (reset state). Đặt ở root layout hoặc quanh những vùng UI dễ crash.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    if (this.props.onError) this.props.onError(error, errorInfo);
    // eslint-disable-next-line no-console
    console.error("[ErrorBoundary]", error, errorInfo);
  }

  private reset = () => {
    this.setState({ hasError: false, error: undefined });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <div className="flex flex-col items-center justify-center text-center py-16 px-6 border border-destructive/30 rounded-xl bg-destructive/5">
          <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center text-destructive mb-4">
            <AlertCircle className="w-6 h-6" />
          </div>
          <h2 className="text-lg font-semibold mb-1">
            {this.props.title ?? "Đã xảy ra lỗi"}
          </h2>
          <p className="text-sm text-muted-foreground max-w-md mb-4">
            {this.state.error?.message ?? "Vui lòng thử lại sau."}
          </p>
          <Button onClick={this.reset} className="gap-2">
            <RefreshCw className="w-4 h-4" />
            Thử lại
          </Button>
        </div>
      );
    }
    return this.props.children;
  }
}
