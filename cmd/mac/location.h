#import <Foundation/Foundation.h>
#import <cocoa/cocoa.h>
#import <CoreLocation/CoreLocation.h>

int nsnumber2int(NSNumber*);
int run(int);
char* nsstring2cstring(NSString*);

@interface Handler : NSObject{
    void** handle;
  }
  - (void)withHandle:(int)handle;
  - (void)logLonLat:(CLLocation*)location;
  - (void)locationManager:(CLLocationManager *)manager
      didUpdateToLocation:(CLLocation *)newLocation fromLocation:(CLLocation *)oldLocation;
  - (void)locationManager:(CLLocationManager *)manager didFailWithError:(NSError *)error;
@end

typedef struct Coords{
  double lat;
  double lon;
} Coords;

extern void goWithError(int, char *); // Go
extern void goWithCoords(int, Coords *); // Go
extern int QuietLog (FILE *, NSString *format, ...); // http://cocoaheads.byu.edu/wiki/different-nslog
#define QuietDebug(...) (QuietLog (stderr, __VA_ARGS__))
#define QuietError(...) QuietLog (stderr, __VA_ARGS__)
#define printf(...) QuietLog(stdout, __VA_ARGS__)