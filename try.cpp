#include <string>
#include <iostream>

using namespace std;
void say(string what) {
    cout << what ;
}
void say(int what) {
    cout << to_string(what) ;
}