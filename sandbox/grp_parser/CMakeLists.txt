cmake_minimum_required(VERSION 3.25)
project(grp_parser)

set(CMAKE_CXX_STANDARD 17)

# 引入 nlohmann/json
include(FetchContent)
FetchContent_Declare(
        json
        URL https://github.com/nlohmann/json/releases/download/v3.11.3/json.tar.xz
)
FetchContent_MakeAvailable(json)

# 建立執行檔
add_executable(grp_parser
        grp_parser.cpp
        JsonParser.cpp
)

# 連結 JSON 庫
target_link_libraries(grp_parser PRIVATE nlohmann_json::nlohmann_json)
