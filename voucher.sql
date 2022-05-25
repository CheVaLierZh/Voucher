use voucher;
CREATE TABLE IF NOT EXISTS voucher (
    code VARCHAR(20) PRIMARY KEY,
    seq INT,
    usr VARCHAR(15),
    used BOOLEAN,
    INDEX (used, seq),
    INDEX (usr, seq)
)ENGINE=InnoDB DEFAULT CHARSET=utf8;