CREATE TABLE IF NOT EXISTS aws_cost_history (
    date  DATE           PRIMARY KEY,
    total DECIMAL(10, 4) NOT NULL
);
