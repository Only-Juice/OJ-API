#include <iostream>
#include <nlohmann/json.hpp>
#include <filesystem>
#include <cmath>
#include "JsonParser.hpp"

using JSON = nlohmann::json;
using namespace std;

void Writescore(double score){
    const string file = "../score.txt";
    double old_score = 0.0;
    if (std::filesystem::exists(file)) {
        std::ifstream infile(file);
        std::string line;
        if (std::getline(infile, line)) {
            std::istringstream iss(line);
            iss >> old_score;
        }
        infile.close();
    }
    double new_score = max(score,old_score);
    if(new_score != old_score){
        std::ofstream outfile(file);
        outfile << std::fixed << std::setprecision(2) << new_score << std::endl;
        outfile.close();
    }
}

int main(int argc, char* argv[]) {
    if (argc != 3) {
        cerr << "Usage: ./grp_parser <gtest.json> <score.json>" << "\n";
    }
    JSONParser parser(argv[1],argv[2]);
    parser.Parse();
    cout<< parser.Getscore() <<"\n";
    Writescore(parser.Getscore());


    return 0;
}
