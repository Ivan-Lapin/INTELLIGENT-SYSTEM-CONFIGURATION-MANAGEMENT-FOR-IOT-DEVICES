from training_pipeline import run_training_pipeline


def train():
    leaderboard, metadata = run_training_pipeline()
    print("Training finished.")
    print("Leaderboard:")
    for row in leaderboard:
        print(row)
    print("Best model:")
    print(metadata)


if __name__ == "__main__":
    train()