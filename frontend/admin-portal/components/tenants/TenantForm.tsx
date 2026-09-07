"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { tenantSchema, type TenantFormData } from "@/lib/schema";
import { useMutation } from "@tanstack/react-query";
import { useToast } from "@/components/ui/use-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { adminApi } from "@/lib/api";
import { useRouter } from "next/navigation";

export function TenantForm({
  initialData,
  onSuccess,
}: {
  initialData?: Partial<TenantFormData>;
  onSuccess?: () => void;
}) {
  const { toast } = useToast();
  const router = useRouter();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<TenantFormData>({
    resolver: zodResolver(tenantSchema),
    defaultValues: initialData,
  });

  const mutation = useMutation({
    mutationFn: (data: TenantFormData) => adminApi.createTenant(data as any),
    onSuccess: () => {
      toast({ title: "Tenant created successfully" });
      onSuccess?.();
      router.refresh();
    },
    onError: () => {
      toast({ title: "Failed to create tenant", variant: "destructive" });
    },
  });

  const onSubmit = (data: TenantFormData) => {
    mutation.mutate(data);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="name">Tenant Name *</Label>
        <Input
          id="name"
          {...register("name")}
          placeholder="My Company"
          className={errors.name ? "border-red-500" : ""}
        />
        {errors.name && (
          <p className="text-xs text-red-500">{errors.name.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="slug">Slug *</Label>
        <Input
          id="slug"
          {...register("slug")}
          placeholder="my-company"
          className={errors.slug ? "border-red-500" : ""}
        />
        {errors.slug && (
          <p className="text-xs text-red-500">{errors.slug.message}</p>
        )}
        <p className="text-xs text-gray-500">
          This will be used in the URL: yourapp.com/[slug]
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="email">Email *</Label>
        <Input
          id="email"
          type="email"
          {...register("email")}
          placeholder="admin@company.com"
          className={errors.email ? "border-red-500" : ""}
        />
        {errors.email && (
          <p className="text-xs text-red-500">{errors.email.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="phone">Phone</Label>
        <Input
          id="phone"
          type="tel"
          {...register("phone")}
          placeholder="+84 123 456 789"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="domain">Custom Domain</Label>
        <Input
          id="domain"
          {...register("domain")}
          placeholder="app.company.com"
        />
      </div>

      <div className="pt-4 flex justify-end gap-3">
        <Button type="button" variant="outline" onClick={() => router.back()}>
          Cancel
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? "Creating..." : "Create Tenant"}
        </Button>
      </div>
    </form>
  );
}

export default TenantForm;
