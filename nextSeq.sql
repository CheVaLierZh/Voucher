use voucher;
CREATE TABLE IF NOT EXISTS nextSeq (
    id INT PRIMARY KEY,
    seq INT
)ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT IGNORE INTO nextSeq VALUES (0, 0);