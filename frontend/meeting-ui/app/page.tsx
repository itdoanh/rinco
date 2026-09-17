"use client";

import { JoinForm } from "@/components/lobby/JoinForm";

export default function HomePage() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-gradient-to-br from-slate-900 to-slate-800 p-6">
      <div className="w-full max-w-md rounded-2xl bg-white p-8 shadow-2xl">
        <JoinForm />
      </div>
    </main>
  );
}
