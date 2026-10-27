BEGIN;
DELETE FROM districts WHERE slug IN (
    'abseron', 'xacmaz', 'quba', 'qusar', 'siyezen', 'sabran', 'xizi', 'qobustan',
    'samaxi', 'ismayilli', 'qebele', 'oguz', 'seki', 'qax', 'zaqatala', 'balaken',
    'qazax', 'agstafa', 'tovuz', 'semkir', 'gedebey', 'daskesen', 'goygol', 'samux',
    'goranboy', 'yevlax', 'berde', 'terter', 'agdas', 'goycay', 'ucar', 'zerdab',
    'kurdemir', 'haciqabul', 'saatli', 'sabirabad', 'agcabedi', 'imisli', 'beyleqan',
    'lenkeran', 'astara', 'masalli', 'celilabad', 'yardimli', 'lerik', 'salyan',
    'neftcala', 'susa', 'agdam', 'xocali', 'xocavend', 'agdere', 'fuzuli', 'cebrayil',
    'zengilan', 'lacin', 'serur', 'babek', 'culfa', 'ordubad', 'sahbuz', 'kengerli', 'sederek'
);
COMMIT;
