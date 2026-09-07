"use client";

interface StatsProps {
  data: {
    stats?: Array<{
      value: string;
      label: string;
      prefix?: string;
      suffix?: string;
    }>;
    style?: string;
  };
}

export function Stats({ data }: StatsProps) {
  const defaultStats = [
    { value: "7.500", label: "tỷ - 17.000 tỷ đồng/ngày", prefix: "", suffix: "" },
    { value: "TOP", label: "Thị phần MXV Q2/2026", prefix: "", suffix: "1" },
    { value: "080", label: "Thành viên kinh doanh MXV", prefix: "", suffix: "" },
    { value: "50", label: "Ebook miễn phí", prefix: "+", suffix: "" },
  ];

  const stats = data.stats?.length ? data.stats : defaultStats;

  return (
    <section className="bg-gradient-to-r from-navy-900 to-navy-700 py-12">
      <div className="max-w-7xl mx-auto px-4">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
          {stats.map((stat, idx) => (
            <div key={idx} className="text-center">
              <div className="text-3xl md:text-4xl lg:text-5xl font-black text-white mb-2">
                <span className="text-gold">{stat.prefix}</span>
                {stat.value}
                <span className="text-gold">{stat.suffix}</span>
              </div>
              <p className="text-white/70 text-sm md:text-base">{stat.label}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

export default Stats;
