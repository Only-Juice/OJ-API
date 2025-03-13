#!/bin/bash

cmake -B build -G Ninja -DCMAKE_BUILD_TYPE=Debug -DFETCH_GOOGLETEST=OFF
cmake --build build --clean-first

mkdir -p build/grp
mkdir -p build/valgrind

for test in $(find build -name "ut*" ! -name "*.*")
do
    $test --gtest_output=json:"build/grp/$(basename $test).json" > /dev/null; echo Return $?
    valgrind --error-exitcode=2 --track-origins=yes --leak-check=full --log-fd=9 9>"build/valgrind/$(basename $test).log" $test > /dev/null
done
python3 utils/grp_parser.py $CODE_PATH/build/grp/ut_all.json

# {
#     "script": "#!/bin/bash\n\ncmake -B build -G Ninja -DCMAKE_BUILD_TYPE=Debug -DFETCH_GOOGLETEST=OFF\ncmake --build build --clean-first\n\nmkdir -p build/grp\nmkdir -p build/valgrind\n\nfor test in $(find build -name \"ut*\" ! -name \"*.*\")\ndo\n$test --gtest_output=json:\"build/grp/$(basename $test).json\" > /dev/null; echo Return $?\nvalgrind --error-exitcode=2 --track-origins=yes --leak-check=full --log-fd=9 9>\"build/valgrind/$(basename $test).log\" $test > /dev/null\ndone\npython3 utils/grp_parser.py $CODE_PATH/build/grp/ut_all.json",
#     "source_git_url": "zre/OOP2023f_111590028_HW"
# }