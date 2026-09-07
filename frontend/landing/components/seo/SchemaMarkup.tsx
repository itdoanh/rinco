export default function SchemaMarkup() {
  const eventSchema = {
    "@context": "https://schema.org",
    "@type": "Event",
    name: "Webinar Miễn Phí: Tối Ưu Dòng Tiền Với Hàng Hóa Phái Sinh 2026",
    description:
      "Chia sẻ trực tuyến miễn phí qua Zoom - Giải mã đầu tư hàng hóa phái sinh 2026. Tặng Bộ 10 Ebook cho 50 nhà đầu tư đăng ký sớm nhất.",
    startDate: "2026-09-07T20:00:00+07:00",
    endDate: "2026-09-07T22:00:00+07:00",
    eventStatus: "https://schema.org/EventScheduled",
    eventAttendanceMode: "https://schema.org/OnlineEventAttendanceMode",
    location: {
      "@type": "VirtualLocation",
      url: "https://zoom.us",
    },
    organizer: {
      "@type": "Organization",
      name: "Công ty Cổ phần Công nghệ Tài chính APEX",
      url: "https://hanghoaphaisinh.net/",
    },
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "VND",
      availability: "https://schema.org/LimitedAvailability",
    },
  };

  const organizationSchema = {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: "APEX Fintech",
    url: "https://hanghoaphaisinh.net/",
    logo: "https://hanghoaphaisinh.net/assets/img/logo_APEX_K_NEN.webp",
    contactPoint: {
      "@type": "ContactPoint",
      telephone: "+84-984386538",
      contactType: "customer service",
      availableLanguage: "Vietnamese",
    },
    sameAs: [
      "https://www.facebook.com/apexfintech",
      "https://zalo.me/0984386538",
    ],
  };

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(eventSchema) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(organizationSchema) }}
      />
    </>
  );
}
