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


def autolabel(bars):
    """Добавляет числовые значения над столбцами"""
    for bar in bars:
        height = bar.get_height()
        plt.text(
            bar.get_x() + bar.get_width() / 2,
            height,
            f"{height:.4f}",
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

    print(grouped)

    x = range(len(grouped))
    width = 0.35

    # 1. Failure rate
    plt.figure(figsize=(12, 7), dpi=200)

    bars = plt.bar(grouped["scenario"], grouped["failure_rate"])

    autolabel(bars)

    plt.title("Failure Rate by Scenario", fontweight="bold")
    plt.ylabel("Failure Rate (ratio)")
    plt.xlabel("Scenario")
    plt.grid(axis="y", linestyle="--", alpha=0.6)

    plt.tight_layout()
    plt.savefig(OUT_DIR / "failure_rate.png")
    plt.close()

    # 2. Latency before/after
    plt.figure(figsize=(12, 7), dpi=200)

    bars1 = plt.bar(
        [i - width/2 for i in x],
        grouped["avg_latency_before"],
        width=width,
        label="Before deployment"
    )

    bars2 = plt.bar(
        [i + width/2 for i in x],
        grouped["avg_latency_after"],
        width=width,
        label="After deployment"
    )

    autolabel(bars1)
    autolabel(bars2)

    plt.xticks(list(x), grouped["scenario"])
    plt.title("Average Latency Before/After Deployment", fontweight="bold")
    plt.ylabel("Latency (ms)")
    plt.xlabel("Scenario")
    plt.legend()
    plt.grid(axis="y", linestyle="--", alpha=0.6)

    plt.tight_layout()
    plt.savefig(OUT_DIR / "latency_before_after.png")
    plt.close()

    # 3. Packet loss before/after
    plt.figure(figsize=(12, 7), dpi=200)

    bars1 = plt.bar(
        [i - width/2 for i in x],
        grouped["avg_loss_before"],
        width=width,
        label="Before deployment"
    )

    bars2 = plt.bar(
        [i + width/2 for i in x],
        grouped["avg_loss_after"],
        width=width,
        label="After deployment"
    )

    autolabel(bars1)
    autolabel(bars2)

    plt.xticks(list(x), grouped["scenario"])
    plt.title("Average Packet Loss Before/After Deployment", fontweight="bold")
    plt.ylabel("Packet Loss (ratio)")
    plt.xlabel("Scenario")
    plt.legend()
    plt.grid(axis="y", linestyle="--", alpha=0.6)

    plt.tight_layout()
    plt.savefig(OUT_DIR / "loss_before_after.png")
    plt.close()

    print(f"Plots saved to: {OUT_DIR}")


if __name__ == "__main__":
    main()