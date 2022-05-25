use voucher;
CREATE TABLE IF NOT EXISTS activity (
    seq INT PRIMARY KEY,
    startTime TIMESTAMP,
    endTime TIMESTAMP,
    redeemLimit INT,
    total INT,
    rest INT,
    description TEXT
)ENGINE=InnoDB DEFAULT CHARSET=utf8;