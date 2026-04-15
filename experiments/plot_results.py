from pathlib import Path

import matplotlib.pyplot as plt
import pandas as pd

ROOT = Path(__file__).resolve().parent
RESULTS_CSV = ROOT / "results" / "experiment_results.csv"
OUT_DIR = ROOT / "results"
OUT_DIR.mkdir(parents=True, exist_ok=True)

plt.rcParams.update({
    "font.size": 16,
    "axes.titlesize": 22,
    "axes.labelsize": 18,
    "xtick.labelsize": 16,
    "ytick.labelsize": 16,
    "legend.fontsize": 16,
})


SCENARIO_NAMES = {
    "baseline": "Базовый",
    "canary": "Канареечный",
    "smart": "Интеллектуальный",
}


def map_scenario_names(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    df["scenario"] = df["scenario"].map(lambda x: SCENARIO_NAMES.get(x, x))
    return df


def autolabel(bars, decimals: int = 4):
    """Добавляет числовые значения над столбцами"""
    for bar in bars:
        height = bar.get_height()
        plt.text(
            bar.get_x() + bar.get_width() / 2,
            height,
            f"{height:.{decimals}f}",
            ha="center",
            va="bottom",
            fontsize=14,
            fontweight="bold"
        )


def main():
    df = pd.read_csv(RESULTS_CSV)

    grouped = df.groupby("scenario", as_index=False).agg({
        "failure_rate": "mean",
        "avg_latency_before": "mean",
        "avg_latency_after": "mean",
        "avg_loss_before": "mean",
        "avg_loss_after": "mean",
        "avg_jitter_before": "mean",
        "avg_jitter_after": "mean",
    })

    grouped = map_scenario_names(grouped)

    print(grouped)

    x = range(len(grouped))
    width = 0.35

    # 1. Частота неуспешных развертываний
    plt.figure(figsize=(12, 7), dpi=200)

    bars = plt.bar(grouped["scenario"], grouped["failure_rate"])

    autolabel(bars, decimals=4)

    plt.title("Частота неуспешных развертываний по сценариям", fontweight="bold")
    plt.ylabel("Частота отказов, доля")
    plt.xlabel("Сценарий развертывания")
    plt.grid(axis="y", linestyle="--", alpha=0.6)

    plt.tight_layout()
    plt.savefig(OUT_DIR / "failure_rate.png")
    plt.close()

    # 2. Средняя задержка до/после развертывания
    plt.figure(figsize=(12, 7), dpi=200)

    bars1 = plt.bar(
        [i - width / 2 for i in x],
        grouped["avg_latency_before"],
        width=width,
        label="До развертывания"
    )

    bars2 = plt.bar(
        [i + width / 2 for i in x],
        grouped["avg_latency_after"],
        width=width,
        label="После развертывания"
    )

    autolabel(bars1, decimals=2)
    autolabel(bars2, decimals=2)

    plt.xticks(list(x), grouped["scenario"])
    plt.title("Средняя задержка до и после развертывания", fontweight="bold")
    plt.ylabel("Задержка, мс")
    plt.xlabel("Сценарий развертывания")
    plt.legend()
    plt.grid(axis="y", linestyle="--", alpha=0.6)

    plt.tight_layout()
    plt.savefig(OUT_DIR / "latency_before_after.png")
    plt.close()

    # 3. Средняя packet loss до/после развертывания
    plt.figure(figsize=(12, 7), dpi=200)

    bars1 = plt.bar(
        [i - width / 2 for i in x],
        grouped["avg_loss_before"],
        width=width,
        label="До развертывания"
    )

    bars2 = plt.bar(
        [i + width / 2 for i in x],
        grouped["avg_loss_after"],
        width=width,
        label="После развертывания"
    )

    autolabel(bars1, decimals=4)
    autolabel(bars2, decimals=4)

    plt.xticks(list(x), grouped["scenario"])
    plt.title("Средняя доля потерь пакетов до и после развертывания", fontweight="bold")
    plt.ylabel("Потери пакетов, доля")
    plt.xlabel("Сценарий развертывания")
    plt.legend()
    plt.grid(axis="y", linestyle="--", alpha=0.6)

    plt.tight_layout()
    plt.savefig(OUT_DIR / "loss_before_after.png")
    plt.close()

    print(f"Графики сохранены в: {OUT_DIR}")


if __name__ == "__main__":
    main()