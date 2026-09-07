"use client";

import Image from "next/image";

interface LogoCloudProps {
  data: {
    title?: string;
    logos?: Array<{ name: string; image?: string }>;
    speed?: number;
  };
}

export function LogoCloud({ data }: LogoCloudProps) {
  const logos = data.logos?.length
    ? data.logos
    : [
        { name: "MXV" },
        { name: "CQG" },
        { name: "NYMEX" },
        { name: "CBOT" },
        { name: "LME" },
      ];

  // Duplicate for seamless loop
  const allLogos = [...logos, ...logos];

  return (
    <section className="bg-white py-10 md:py-12 border-y border-gray-100">
      <div className="max-w-7xl mx-auto px-4">
        <p className="text-center text-[11px] md:text-xs font-bold uppercase tracking-[0.18em] text-gray-500 mb-6">
          {data.title || "Được cấp phép chính thức và liên thông giao dịch quốc tế"}
        </p>
        <div className="logo-marquee">
          <div
            className="logo-marquee-track"
            style={{
              animationDuration: `${data.speed || 20}s`,
            }}
          >
            {allLogos.map((logo, idx) => (
              <div key={idx} className="logo-chip">
                {logo.image ? (
                  <Image
                    src={logo.image}
                    alt={logo.name}
                    width={80}
                    height={40}
                    className="h-10 w-auto object-contain"
                  />
                ) : (
                  <span className="logo-text">{logo.name}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

export default LogoCloud;
