#import <cocoa/cocoa.h>
#import <CoreLocation/CoreLocation.h>
#import "location.h"
#include <unistd.h>

int
nsnumber2int(NSNumber *i) {
    if (i == NULL) { return 0; }
    return i.intValue;
}
char*
nsstring2cstring(NSString *s) {
    if (s == NULL) { return NULL; }

    char *cstr = [s UTF8String];
    return cstr;
}


@implementation Handler
  - (void)withHandle:(int)h
  {
    handle = h;
  }
  - (void)logLonLat:(CLLocation*)location
  {
    CLLocationCoordinate2D coordinate = location.coordinate;
    QuietDebug ([location description]);
    QuietDebug (@"\n");

    printf(@"timestamp: %@\n", location.timestamp);
    printf(@"latitude,longitude: %f,%f\n", coordinate.latitude, coordinate.longitude);
    printf(@"altitude: %f\n", location.altitude);
    printf(@"horizontalAccuracy: %f\n", location.horizontalAccuracy);
    printf(@"verticalAccuracy: %f\n", location.verticalAccuracy);
    printf(@"speed: %f\n", location.speed);
    printf(@"course: %f\n", location.course);

    Coords c;
    c.lat = coordinate.latitude;
    c.lon = coordinate.longitude;

    goWithCoords(handle, &c);
  }

  - (void)locationManager:(CLLocationManager *)manager
      didUpdateToLocation:(CLLocation *)newLocation fromLocation:(CLLocation *)oldLocation {
      NSAutoreleasePool *pool = [[NSAutoreleasePool alloc] init];
      [self logLonLat:newLocation];
      [pool drain];
  }

  - (void)locationManager:(CLLocationManager *)manager didFailWithError:(NSError *)error{
      QuietError(@"Error: %@", error.localizedDescription);
      goWithError(handle, nsstring2cstring(error.localizedDescription));
  }
@end


int QuietLog (FILE *stream, NSString *format, ...)
{
    if (format == nil) {
        fprintf(stream, "nil\n");
        return -1;
    }
    // Get a reference to the arguments that follow the format parameter
    va_list argList;
    va_start(argList, format);
    // Perform format string argument substitution, reinstate %% escapes, then print
    NSString *s = [[NSString alloc] initWithFormat:format arguments:argList];
    fprintf(stream, "%s\n", [[s stringByReplacingOccurrencesOfString:@"%%" withString:@"%%%%"] UTF8String]);
    [s release];
    va_end(argList);
    return 0;
}

int run(int h) {
  id obj = [[Handler alloc] init];
  [obj withHandle:h];
  id lm = nil;
  TRY_AGAIN:
  if ([CLLocationManager locationServicesEnabled]) {
    QuietDebug(@"location service enabled\n");
    lm = [[CLLocationManager alloc] init];
    [lm setDelegate:obj];
    [lm startUpdatingLocation];
  }
  else {
    QuietDebug(@"location service disabled\n");
    goWithError(h, nsstring2cstring(@"location service disabled"));
    sleep(1);
    goto TRY_AGAIN;
  }

  CFRunLoopRun();
  [lm release];
  [obj release];
  return 0;
}